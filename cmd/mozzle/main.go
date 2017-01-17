// mozzle is a command-line utility which subscribes collects Cloud Foundry
// application events and emits them to Riemann.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/Bo0mer/mozzle"
	"github.com/pkg/errors"
)

var (
	apiAddr      string
	insecure     bool
	username     string
	password     string
	accessToken  string
	refreshToken string
	org          string
	space        string

	riemannAddr string

	eventsTTL     float64
	queueSize     int
	reportVersion bool
)

// populated using -ldflags.
var (
	version    string
	build      string
	buildstamp string
)

func init() {
	flag.StringVar(&apiAddr, "api", "https://api.bosh-lite.com", "Address of the Cloud Foundry API")
	flag.BoolVar(&insecure, "insecure", false, "Please, please, don't!")
	flag.StringVar(&username, "username", "", "Cloud Foundry user; usage is discouraged - see token option instead")
	flag.StringVar(&password, "password", "", "Cloud Foundry password; usage is discouraged - see token option instead")
	flag.StringVar(&accessToken, "access-token", "", "Cloud Foundry OAuth2 token; either token or username and password must be provided")
	flag.StringVar(&refreshToken, "refresh-token", "", "Cloud Foundry OAuth2 refresh token; to be used with the token flag")
	flag.StringVar(&org, "org", "NASA", "Cloud Foundry organization")
	flag.StringVar(&space, "space", "rocket", "Cloud Foundry space")

	flag.StringVar(&riemannAddr, "riemann", "127.0.0.1:5555", "Address of the Riemann endpoint")

	flag.Float64Var(&eventsTTL, "events-ttl", 30.0, "TTL for emitted events (in seconds)")
	flag.IntVar(&queueSize, "events-queue-size", 256, "Queue size for outgoing events")
	flag.BoolVar(&reportVersion, "v", false, "Report mozzle version")
	flag.BoolVar(&reportVersion, "version", false, "Report mozzle version")
}

func main() {
	flag.Parse()
	if reportVersion {
		printVersion()
		os.Exit(0)
	}
	riemann := &mozzle.RiemannEmitter{}
	riemann.Initialize(riemannAddr, float32(eventsTTL), queueSize)
	defer func() {
		if err := riemann.Close(); err != nil {
			fmt.Printf("mozzle: error closing riemann emitter: %v\n", err)
		}
	}()

	var token *oauth2.Token
	if accessToken != "" {
		var err error
		token, err = parseToken(accessToken, refreshToken)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mozzle: error parsing token: %v\n", err)
			os.Exit(1)
		}
	}
	t := mozzle.Target{
		API:      apiAddr,
		Username: username,
		Password: password,
		Token:    token,
		Insecure: insecure,
		Org:      org,
		Space:    space,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		<-sig
		fmt.Println("exiting...")
		cancel()
	}()

	if err := mozzle.Monitor(ctx, t, riemann); err != nil {
		fmt.Printf("mozzle: error occured during Monitor: %v\n", err)
	}
}

func printVersion() {
	fmt.Printf("mozzle version %s build %s at %s\n", version, build, buildstamp)
}

func parseToken(accessToken, refreshToken string) (*oauth2.Token, error) {
	if strings.HasPrefix(accessToken, "bearer ") {
		accessToken = accessToken[len("bearer "):]
	}
	token, err := parseBearerToken(accessToken)
	if err != nil {
		return nil, err
	}
	token.RefreshToken = refreshToken
	return token, nil
}

// parseBearerToken converts the string s to an OAuth2 bearer token.
// It must be of the form <header>.<payload>.<signature>, where header,
// payload and signature are base64 encoded JSON objects.
func parseBearerToken(s string) (*oauth2.Token, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token syntax")
	}
	claims, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Wrap(err, "error decoding token claims segment")
	}
	var t struct {
		Exp int64 `json:"exp"`
	}
	err = json.Unmarshal([]byte(claims), &t)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding token claims")
	}
	return &oauth2.Token{
		AccessToken: s,
		TokenType:   "bearer",
		Expiry:      time.Unix(t.Exp, 0),
	}, nil
}
