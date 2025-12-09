package beaconauth_test

import (
	"fmt"
	"log"

	beaconauth "github.com/marshallshelly/beacon-auth"
	"github.com/marshallshelly/beacon-auth/adapters/memory"
)

type SilentLogger struct{}

func (l *SilentLogger) Debug(msg string, fields ...interface{}) {}
func (l *SilentLogger) Info(msg string, fields ...interface{})  {}
func (l *SilentLogger) Warn(msg string, fields ...interface{})  {}
func (l *SilentLogger) Error(msg string, fields ...interface{}) {}

func ExampleNew() {
	// Initialize memory adapter
	adapter := memory.New()

	// Initialize BeaconAuth with silent logger to avoid random output in test
	authInstance, err := beaconauth.New(
		beaconauth.WithAdapter(adapter),
		beaconauth.WithSecret("test-secret-key-must-be-32-bytes-long!"),
		beaconauth.WithBaseURL("http://localhost:3000"),
		beaconauth.WithLogger(&SilentLogger{}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer authInstance.Close()

	fmt.Println("BeaconAuth initialized successfully")

	// Output:
	// BeaconAuth initialized successfully
}
