package integration

import (
	"encoding/json"
	"fmt"
	"regexp"

	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
	"github.com/tidwall/buntdb"
)

type (
	Inmemory struct {
		DB *buntdb.DB
	}
)

func initInMemoryStorage() IntegrationStorage {
	db, _ := buntdb.Open(":memory:")
	return &Inmemory{
		DB: db,
	}
}

func (d *Inmemory) Insert(i Integration, logger cm_logger.LoggingFn) error {
	str, err := integrationToString(&i)
	if err != nil {
		return err
	}
	return d.DB.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(i.Metadata.Name, str, nil)
		return err
	})
}

func integrationToString(i *Integration) (string, error) {
	b, err := json.Marshal(i)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func stringToIntegration(str string) (*Integration, error) {
	result := &Integration{}
	b := []byte(str)
	err := json.Unmarshal(b, result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (d *Inmemory) Find(name string, logger cm_logger.LoggingFn) (*Integration, error) {
	result := &Integration{}
	err := d.DB.View(func(tx *buntdb.Tx) error {
		val, err := tx.Get(name)
		result, err = stringToIntegration(val)
		if err != nil {
			return err
		}
		logMessage := "Found integration"
		logger(cm_logger.DebugLevel, logMessage, "Name", result.Metadata.Name)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (d *Inmemory) GetIntegrationsByResourceName(repo string, logger cm_logger.LoggingFn) ([]*Integration, error) {
	logMessage := "GetIntegrationsByResourceName"
	logger(cm_logger.DebugLevel, logMessage, "Repo", repo)
	result := []*Integration{}
	err := d.DB.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend("", func(key, value string) bool {
			integration, _ := stringToIntegration(value)
			resource := integration.Spec.Resource
			logger(cm_logger.DebugLevel, logMessage, "Resource", resource)
			if resource.Repo == repo {
				result = append(result, integration)
			}
			return true
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
func (d *Inmemory) GetIntegrationsMatchResourceByRegex(repo string, chart string, logger cm_logger.LoggingFn) ([]*Integration, error) {
	logMessage := "GetIntegrationsMatchResourceByRegex"
	logger(cm_logger.DebugLevel, logMessage, "Repo", repo, "Chart", chart)
	result := []*Integration{}
	err := d.DB.View(func(tx *buntdb.Tx) error {
		err := tx.Ascend("", func(key, value string) bool {
			integration, _ := stringToIntegration(value)
			logger(cm_logger.DebugLevel, "Checking resource", "Name", integration.Metadata.Name)
			resource := integration.Spec.Resource
			name := fmt.Sprintf("%s/%s", repo, chart)
			pattern := resource.Repo
			if resource.Chart != "" {
				pattern += fmt.Sprintf("/%s", resource.Chart)
			}
			logger(cm_logger.DebugLevel, "Required match", "Pattern", pattern)
			res, _ := regexp.MatchString(pattern, name)
			if res {
				logger(cm_logger.DebugLevel, "Matched!")
				result = append(result, integration)
			}
			return true
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (d *Inmemory) Remove(name string, logger cm_logger.LoggingFn) error {
	err := d.DB.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(name)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
