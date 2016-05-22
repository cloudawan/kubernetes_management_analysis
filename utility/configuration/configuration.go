// Copyright 2015 CloudAwan LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configuration

import (
	"errors"
	analysisLogger "github.com/cloudawan/cloudone_analysis/utility/logger"
	"github.com/cloudawan/cloudone_utility/configuration"
	"github.com/cloudawan/cloudone_utility/logger"
	"github.com/cloudawan/cloudone_utility/restclient"
	"io/ioutil"
	"time"
)

var log = analysisLogger.GetLogManager().GetLogger("utility")

var configurationContent = `
{
	"certificate": "/etc/cloudone_analysis/development_cert.pem",
	"key": "/etc/cloudone_analysis/development_key.pem",
	"restapiPort": 8082,
	"elasticsearchHost": ["127.0.0.1"],
	"elasticsearchPort": 9200,
	"kubeApiServerEndPoints": ["https://kubernetes.default.svc.cluster.local:443"],
	"kubeApiServerHealthCheckTimeoutInMilliSecond": 1000,
	"kubeApiServerTokenPath": "/var/run/secrets/kubernetes.io/serviceaccount/token",
	"singletonLockTimeoutInMilliSecond": 5000,
	"singletonLockWaitingAfterBeingCandidateInMilliSecond": 5000,
	"cloudoneProtocol": "https",
	"cloudoneHost": "127.0.0.1",
	"cloudonePort": 8081
}
`

var LocalConfiguration *configuration.Configuration

const (
	KubeApiServerHealthCheckTimeoutInMilliSecond = 1000
)

func init() {
	err := Reload()
	if err != nil {
		log.Critical(err)
		panic(err)
	}
}

func Reload() error {
	localConfiguration, err := configuration.CreateConfiguration("cloudone_analysis", configurationContent)
	if err == nil {
		LocalConfiguration = localConfiguration
	}

	return err
}

func GetAvailablekubeApiServerEndPoint() (returnedEndPoint string, returnedToken string, returnedError error) {
	defer func() {
		if err := recover(); err != nil {
			returnedEndPoint = ""
			returnedToken = ""
			returnedError = err.(error)
			log.Error("GetAvailablekubeApiServerEndPoint Error: %s", err)
			log.Error(logger.GetStackTrace(4096, false))
		}
	}()

	kubeApiServerEndPointSlice, ok := LocalConfiguration.GetStringSlice("kubeApiServerEndPoints")
	if ok == false {
		log.Error("Fail to get configuration kubeApiServerEndPoints")
		return "", "", errors.New("Fail to get configuration kubeApiServerEndPoints")
	}

	kubeApiServerTokenPath, ok := LocalConfiguration.GetString("kubeApiServerTokenPath")
	if ok == false {
		log.Error("Fail to get configuration kubeApiServerTokenPath")
		return "", "", errors.New("Fail to get configuration kubeApiServerTokenPath")
	}

	fileContent, err := ioutil.ReadFile(kubeApiServerTokenPath)
	if err != nil {
		log.Error("Fail to get the file content of kubeApiServerTokenPath %s", kubeApiServerTokenPath)
		return "", "", errors.New("Fail to get the file content of kubeApiServerTokenPath " + kubeApiServerTokenPath)
	}

	kubeApiServerHealthCheckTimeoutInMilliSecond, ok := LocalConfiguration.GetInt("kubeApiServerHealthCheckTimeoutInMilliSecond")
	if ok == false {
		kubeApiServerHealthCheckTimeoutInMilliSecond = KubeApiServerHealthCheckTimeoutInMilliSecond
	}

	token := "Bearer " + string(fileContent)
	headerMap := make(map[string]string)
	headerMap["Authorization"] = token

	for _, kubeApiServerEndPoint := range kubeApiServerEndPointSlice {
		result, err := restclient.HealthCheck(
			kubeApiServerEndPoint,
			headerMap,
			time.Duration(kubeApiServerHealthCheckTimeoutInMilliSecond)*time.Millisecond)

		if result {
			return kubeApiServerEndPoint, token, nil
		} else {
			if err != nil {
				log.Error(err)
			}
		}
	}

	log.Error("No available kube apiserver endpoint")
	return "", "", errors.New("No available kube apiserver endpoint")
}
