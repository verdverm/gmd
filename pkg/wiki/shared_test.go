//go:build integration

package wiki

import (
	"testing"

	"github.com/verdverm/gmd/pkg/llm"
	"github.com/verdverm/gmd/pkg/ts"
)

type tapeClients struct {
	TS       *ts.Client
	Embedder llm.Embedder
	Chat     llm.ChatModel
	Registry *llm.Registry
	Stop     func()
}

func tapeTest(t *testing.T, tapeFile string) tapeClients {
	t.Helper()
	tape := maybeNewTape(t, tapeFile)
	if tape != nil {
		tape.Start()
		reg := buildTapedRegistry(t, tape)
		return tapeClients{
			TS:       buildTapedTSCWikiClient(t, tape),
			Embedder: reg.Embedder(),
			Chat:     reg.Model(llm.RoleGeneralBig),
			Registry: reg,
			Stop: func() {
				if err := tape.Stop(); err != nil {
					t.Fatal(err)
				}
			},
		}
	}
	return tapeClients{
		TS:       testTSClient,
		Embedder: testRegistry.Embedder(),
		Chat:     testRegistry.Model(llm.RoleGeneralBig),
		Registry: testRegistry,
		Stop:     func() {},
	}
}
