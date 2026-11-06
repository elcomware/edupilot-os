// Package tests provides the Goravel test case for framework-facing feature suites.
package tests

import (
	"github.com/elcomware/edupilot/bootstrap"
	goraveltesting "github.com/goravel/framework/testing"
	"sync"
)

// Boot is explicit so importing a fixture never connects or migrates a database.
var Boot = sync.OnceFunc(func() { bootstrap.Boot() })

type TestCase struct{ goraveltesting.TestCase }
