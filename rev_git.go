package revgit

import (
	"encoding/json"
	"fmt"

	revmongo "github.com/QFO6/rev-mongo"
)

var (
	GitUrl      string
	GitUser     string
	GitPass     string
	GitGrpcUrl  string
	GitToken    string
	GitUtilName = "GitConfig"
)

func Init(configUtil *revmongo.Utils) {
	fmt.Println("Initializing rev-git module...")
	if configUtil.Value == "" {
		fmt.Printf("No valid %s value configured\n", GitUtilName)
		return
	}

	gitConfigObj := map[string]string{}
	err := json.Unmarshal([]byte(configUtil.Value), &gitConfigObj)
	if err != nil {
		fmt.Printf("Error: Parse %s util json value error %v\n", GitUtilName, err)
		return
	}

	var found bool
	if GitGrpcUrl, found = gitConfigObj["grpcUrl"]; !found {
		fmt.Printf("no grpcUrl found in %s util value\n", GitUtilName)
	}

	if GitUrl, found = gitConfigObj["gitUrl"]; !found {
		fmt.Printf("no gitUrl found in %s util value\n", GitUtilName)
	}

	// Generating the git token from git/gitea user settings/security tab
	if GitToken, found = gitConfigObj["gitToken"]; !found {
		fmt.Printf("no gitToken found in %s util value\n", GitUtilName)
		if GitUser, found = gitConfigObj["gitUser"]; !found {
			fmt.Printf("no gitUser found in %s util value\n", GitUtilName)
		}

		if GitPass, found = gitConfigObj["gitPass"]; !found {
			fmt.Printf("no gitPass found in %s util value\n", GitUtilName)
		}
	}

	fmt.Println("rev-git module initialized successfully")
}
