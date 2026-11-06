package providers

import (
	"github.com/elcomware/edupilot/modules/auth/database/seeders"
	"github.com/goravel/framework/contracts/database/seeder"
)

func Seeders() []seeder.Seeder {
	return []seeder.Seeder{&seeders.SchoolAdminSeeder{}}
}
