package config

import "os"

const FF_REPO_VIEW = "FF_REPO_VIEW"

const FF_MOCK_DATA = "FF_MOCK_DATA"

const FF_FULLSEND_INTEGRATION = "FF_FULLSEND_INTEGRATION"

func IsFeatureEnabled(name string) bool {
	_, ok := os.LookupEnv(name)
	return ok
}
