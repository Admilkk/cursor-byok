package forwarder

import (
	"testing"

	"cursor/gen/aiserverv1/aiserverv1connect"
)

func TestAIHandlerTracksLocallyImplementedPaths(t *testing.T) {
	handler := newAIHandler(&Service{})
	if !handler.HandlesPath(aiserverv1connect.AiServiceCountTokensProcedure) {
		t.Fatalf("expected %q to be handled locally", aiserverv1connect.AiServiceCountTokensProcedure)
	}
	if !handler.HandlesPath(dashboardServiceGetTokenUsageProcedure) {
		t.Fatalf("expected %q to be handled locally", dashboardServiceGetTokenUsageProcedure)
	}
	if handler.HandlesPath("/aiserver.v1.AiService/UnknownProcedure") {
		t.Fatal("unknown AI procedure must fall through to upstream")
	}
}
