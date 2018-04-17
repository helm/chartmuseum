package integration

import (
	cm_logger "github.com/kubernetes-helm/chartmuseum/pkg/chartmuseum/logger"
)

var (
	integrationKind    = "Integration"
	integrationVersion = "v1"
	DAL                *Storage
)

func InitStorage() {
	storage := initInMemoryStorage()
	DAL = &Storage{
		db: storage,
	}
}

type (
	IntegrationStorage interface {
		Insert(Integration, cm_logger.LoggingFn) error
		Find(string, cm_logger.LoggingFn) (*Integration, error)
		GetIntegrationsByResourceName(string, cm_logger.LoggingFn) ([]*Integration, error)
		GetIntegrationsMatchResourceByRegex(string, string, cm_logger.LoggingFn) ([]*Integration, error)
		Remove(string, cm_logger.LoggingFn) error
	}

	Database struct {
		IntegrationIDTable map[string]Integration
	}

	Storage struct {
		db IntegrationStorage
	}
)

func (s *Storage) Insert(i Integration, logger cm_logger.LoggingFn) error {
	return s.db.Insert(i, logger)
}

func (s *Storage) Find(name string, logger cm_logger.LoggingFn) (*Integration, error) {
	return s.db.Find(name, logger)
}

func (s *Storage) GetIntegrationsByResourceName(repo string, logger cm_logger.LoggingFn) ([]*Integration, error) {
	return s.db.GetIntegrationsByResourceName(repo, logger)
}

func (s *Storage) GetIntegrationsMatchResourceByRegex(repo string, chart string, logger cm_logger.LoggingFn) ([]*Integration, error) {
	return s.db.GetIntegrationsMatchResourceByRegex(repo, chart, logger)
}

func (s *Storage) Remove(name string, logger cm_logger.LoggingFn) error {
	return s.db.Remove(name, logger)
}
