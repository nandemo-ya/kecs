package postgres_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPostgres(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Postgres Storage Suite")
}

var _ = BeforeSuite(func() {
	// Start PostgreSQL container once for all tests
	setupPostgresContainer()
})

var _ = AfterSuite(func() {
	// Clean up PostgreSQL container after all tests
	teardownPostgresContainer()
})
