package gui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/jesseduffield/lazygit/pkg/common"
	"github.com/jesseduffield/lazygit/pkg/config"
	"github.com/jesseduffield/lazygit/pkg/gui/controllers/helpers"
	"github.com/sirupsen/logrus"
)

func TestForkBackgroundRoutinesConfigGates(t *testing.T) {
	testCases := []struct {
		name            string
		configure       func(*config.UserConfig)
		wantLogMessages []string
	}{
		{
			name: "fork defaults retain external change polling only",
			wantLogMessages: []string{
				"externalChangeCheckInterval' (0) is invalid, disabling external change detection",
			},
		},
		{
			name: "all automation opted in",
			configure: func(userConfig *config.UserConfig) {
				userConfig.Git.AutoFetch = true
				userConfig.Git.AutoRefresh = true
				userConfig.Git.AutoDetectExternalChanges = true
			},
			wantLogMessages: []string{
				"fetchInterval' (0) is invalid, disabling auto-fetch",
				"refreshInterval' (0) is invalid, disabling auto-refresh",
				"externalChangeCheckInterval' (0) is invalid, disabling external change detection",
			},
		},
		{
			name: "all automation disabled",
			configure: func(userConfig *config.UserConfig) {
				userConfig.Git.AutoFetch = false
				userConfig.Git.AutoRefresh = false
				userConfig.Git.AutoDetectExternalChanges = false
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			userConfig := config.GetDefaultConfig()
			userConfig.Refresher.FetchInterval = 0
			userConfig.Refresher.RefreshInterval = 0
			userConfig.Refresher.ExternalChangeCheckInterval = 0
			if testCase.configure != nil {
				testCase.configure(userConfig)
			}

			var logOutput bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&logOutput)
			logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true, DisableColors: true})

			commonState := common.NewDummyCommon()
			commonState.SetUserConfig(userConfig)
			commonState.Log = logrus.NewEntry(logger)
			gui := &Gui{
				Common: commonState,
				Config: config.NewDummyAppConfig(),
				c:      &helpers.HelperCommon{Common: commonState},
			}
			manager := &BackgroundRoutineMgr{gui: gui}

			// Invalid intervals expose each enabled startup gate without spawning workers.
			manager.startBackgroundRoutines()
			manager.triggerImmediateFetch()

			if manager.triggerFetch != nil {
				t.Fatalf("triggerFetch = %v, want nil", manager.triggerFetch)
			}

			output := logOutput.String()
			for _, message := range testCase.wantLogMessages {
				if !strings.Contains(output, message) {
					t.Errorf("log output %q does not contain %q", output, message)
				}
			}

			wantLogCount := len(testCase.wantLogMessages)
			if got := strings.Count(output, "level=error"); got != wantLogCount {
				t.Errorf("error log count = %d, want %d; output: %q", got, wantLogCount, output)
			}
		})
	}
}
