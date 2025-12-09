package schema

import (
	"fmt"
	"strings"
)

// Config holds generation configuration
type Config struct {
	Adapter string
	Plugins []string
	IDType  string // "string", "uuid", "serial"
}

// GenerateSQL generates the SQL schema based on config
func GenerateSQL(cfg *Config) (string, error) {
	var sqlBuilder strings.Builder

	// 1. Generate Core Tables
	coreSQL, err := generateCore(cfg)
	if err != nil {
		return "", err
	}
	sqlBuilder.WriteString("-- Core Schema\n")
	sqlBuilder.WriteString(coreSQL)
	sqlBuilder.WriteString("\n")

	// 2. Generate Plugin Tables
	for _, plugin := range cfg.Plugins {
		pluginSQL, err := generatePlugin(plugin, cfg)
		if err != nil {
			return "", err
		}
		if pluginSQL != "" {
			sqlBuilder.WriteString(fmt.Sprintf("-- Plugin: %s\n", plugin))
			sqlBuilder.WriteString(pluginSQL)
			sqlBuilder.WriteString("\n")
		}
	}

	return sqlBuilder.String(), nil
}

func generateCore(cfg *Config) (string, error) {
	switch cfg.Adapter {
	case "postgres":
		return generatePostgresCore(cfg.IDType), nil
	case "mysql":
		return generateMySQLCore(cfg.IDType), nil
	case "sqlite":
		return generateSQLiteCore(cfg.IDType), nil
	case "mssql":
		return generateMSSQLCore(cfg.IDType), nil
	default:
		return "", fmt.Errorf("unsupported adapter: %s", cfg.Adapter)
	}
}

func generatePlugin(plugin string, cfg *Config) (string, error) {
	switch plugin {
	case "twofa":
		switch cfg.Adapter {
		case "postgres":
			return generatePostgresTwoFA(cfg.IDType), nil
		case "mysql":
			return generateMySQLTwoFA(cfg.IDType), nil
		case "sqlite":
			return generateSQLiteTwoFA(cfg.IDType), nil
		case "mssql":
			return generateMSSQLTwoFA(cfg.IDType), nil
		}
	case "emailpassword", "oauth":
		return "", nil // No extra tables needed, uses 'accounts'
	}
	return "", nil
}

// --- Postgres ---

func generatePostgresCore(idType string) string {
	idDef := "VARCHAR(255) PRIMARY KEY"
	if idType == "uuid" {
		idDef = "UUID PRIMARY KEY DEFAULT gen_random_uuid()"
	} else if idType == "serial" {
		idDef = "SERIAL PRIMARY KEY"
	}

	// For references, if ID is serial (int), foreign keys must be int
	fkDef := "VARCHAR(255)"
	if idType == "uuid" {
		fkDef = "UUID"
	} else if idType == "serial" {
		fkDef = "INTEGER"
	}

	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS users (
    id %s,
    email VARCHAR(255) NOT NULL UNIQUE,
    email_verified BOOLEAN DEFAULT FALSE,
    name VARCHAR(255),
    image TEXT,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    role VARCHAR(50),
    banned BOOLEAN DEFAULT FALSE,
    ban_reason TEXT,
    ban_expires TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id %s,
    user_id %s NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    impersonated_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS accounts (
    id %s,
    user_id %s NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id VARCHAR(255) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    provider_type VARCHAR(50) NOT NULL,
    password TEXT,
    access_token TEXT,
    refresh_token TEXT,
    access_token_expires_at TIMESTAMP,
    refresh_token_expires_at TIMESTAMP,
    scope TEXT,
    id_token TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider_id, account_id)
);

CREATE TABLE IF NOT EXISTS verifications (
    id %s,
    identifier VARCHAR(255) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`, idDef, idDef, fkDef, idDef, fkDef, idDef)
}

func generatePostgresTwoFA(idType string) string {
	idDef := "VARCHAR(255) PRIMARY KEY"
	if idType == "uuid" {
		idDef = "UUID PRIMARY KEY DEFAULT gen_random_uuid()"
	} else if idType == "serial" {
		idDef = "SERIAL PRIMARY KEY"
	}

	// For references
	fkDef := "VARCHAR(255)"
	if idType == "uuid" {
		fkDef = "UUID"
	} else if idType == "serial" {
		fkDef = "INTEGER"
	}

	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS two_factors (
    id %s,
    user_id %s NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret TEXT NOT NULL,
    uri TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS two_factor_backup_codes (
    id %s,
    user_id %s NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`, idDef, fkDef, idDef, fkDef)
}

// --- MySQL ---

func generateMySQLCore(idType string) string {
	idDef := "VARCHAR(255) PRIMARY KEY"
	if idType == "uuid" {
		idDef = "CHAR(36) PRIMARY KEY"
	} else if idType == "serial" {
		idDef = "INT AUTO_INCREMENT PRIMARY KEY"
	}

	fkDef := "VARCHAR(255)"
	if idType == "uuid" {
		fkDef = "CHAR(36)"
	} else if idType == "serial" {
		fkDef = "INT"
	}

	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS users (
    id %s,
    email VARCHAR(255) NOT NULL UNIQUE,
    email_verified BOOLEAN DEFAULT FALSE,
    name VARCHAR(255),
    image TEXT,
    two_factor_enabled BOOLEAN DEFAULT FALSE,
    role VARCHAR(50),
    banned BOOLEAN DEFAULT FALSE,
    ban_reason TEXT,
    ban_expires TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id %s,
    user_id %s NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    impersonated_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS accounts (
    id %s,
    user_id %s NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    provider_id VARCHAR(255) NOT NULL,
    provider_type VARCHAR(50) NOT NULL,
    password TEXT,
    access_token TEXT,
    refresh_token TEXT,
    access_token_expires_at TIMESTAMP NULL,
    refresh_token_expires_at TIMESTAMP NULL,
    scope TEXT,
    id_token TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY provider_account (provider_id, account_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS verifications (
    id %s,
    identifier VARCHAR(255) NOT NULL,
    token VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
`, idDef, idDef, fkDef, idDef, fkDef, idDef)
}

func generateMySQLTwoFA(idType string) string {
	idDef := "VARCHAR(255) PRIMARY KEY"
	if idType == "uuid" {
		idDef = "CHAR(36) PRIMARY KEY"
	} else if idType == "serial" {
		idDef = "INT AUTO_INCREMENT PRIMARY KEY"
	}

	fkDef := "VARCHAR(255)"
	if idType == "uuid" {
		fkDef = "CHAR(36)"
	} else if idType == "serial" {
		fkDef = "INT"
	}

	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS two_factors (
    id %s,
    user_id %s NOT NULL,
    secret TEXT NOT NULL,
    uri TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS two_factor_backup_codes (
    id %s,
    user_id %s NOT NULL,
    code VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`, idDef, fkDef, idDef, fkDef)
}

// --- SQLite ---

func generateSQLiteCore(idType string) string {
	// SQLite is simpler, usually INTEGER PRIMARY KEY implies AUTOINCREMENT
	idDef := "TEXT PRIMARY KEY"
	if idType == "serial" {
		idDef = "INTEGER PRIMARY KEY AUTOINCREMENT"
	}

	fkDef := "TEXT"
	if idType == "serial" {
		fkDef = "INTEGER"
	}

	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS users (
    id %s,
    email TEXT NOT NULL UNIQUE,
    email_verified BOOLEAN DEFAULT 0,
    name TEXT,
    image TEXT,
    two_factor_enabled BOOLEAN DEFAULT 0,
    role TEXT,
    banned BOOLEAN DEFAULT 0,
    ban_reason TEXT,
    ban_expires DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id %s,
    user_id %s NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    impersonated_by TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS accounts (
    id %s,
    user_id %s NOT NULL,
    account_id TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    provider_type TEXT NOT NULL,
    password TEXT,
    access_token TEXT,
    refresh_token TEXT,
    access_token_expires_at DATETIME,
    refresh_token_expires_at DATETIME,
    scope TEXT,
    id_token TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider_id, account_id),
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS verifications (
    id %s,
    identifier TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`, idDef, idDef, fkDef, idDef, fkDef, idDef)
}

func generateSQLiteTwoFA(idType string) string {
	idDef := "TEXT PRIMARY KEY"
	if idType == "serial" {
		idDef = "INTEGER PRIMARY KEY AUTOINCREMENT"
	}

	fkDef := "TEXT"
	if idType == "serial" {
		fkDef = "INTEGER"
	}

	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS two_factors (
    id %s,
    user_id %s NOT NULL,
    secret TEXT NOT NULL,
    uri TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS two_factor_backup_codes (
    id %s,
    user_id %s NOT NULL,
    code TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
`, idDef, fkDef, idDef, fkDef)
}

// --- MSSQL ---

func generateMSSQLCore(idType string) string {
	idDef := "NVARCHAR(255) PRIMARY KEY"
	if idType == "uuid" {
		idDef = "UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWID()"
	} else if idType == "serial" {
		idDef = "INT IDENTITY(1,1) PRIMARY KEY"
	}

	fkDef := "NVARCHAR(255)"
	if idType == "uuid" {
		fkDef = "UNIQUEIDENTIFIER"
	} else if idType == "serial" {
		fkDef = "INT"
	}

	return fmt.Sprintf(`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='users' AND xtype='U')
CREATE TABLE users (
    id %s,
    email NVARCHAR(255) NOT NULL UNIQUE,
    email_verified BIT DEFAULT 0,
    name NVARCHAR(255),
    image NVARCHAR(MAX),
    two_factor_enabled BIT DEFAULT 0,
    role NVARCHAR(50),
    banned BIT DEFAULT 0,
    ban_reason NVARCHAR(MAX),
    ban_expires DATETIME2,
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE()
);

IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='sessions' AND xtype='U')
CREATE TABLE sessions (
    id %s,
    user_id %s NOT NULL,
    token NVARCHAR(255) NOT NULL UNIQUE,
    expires_at DATETIME2 NOT NULL,
    ip_address NVARCHAR(45),
    user_agent NVARCHAR(MAX),
    impersonated_by NVARCHAR(255),
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE(),
    CONSTRAINT FK_Session_User FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='accounts' AND xtype='U')
CREATE TABLE accounts (
    id %s,
    user_id %s NOT NULL,
    account_id NVARCHAR(255) NOT NULL,
    provider_id NVARCHAR(255) NOT NULL,
    provider_type NVARCHAR(50) NOT NULL,
    password NVARCHAR(MAX),
    access_token NVARCHAR(MAX),
    refresh_token NVARCHAR(MAX),
    access_token_expires_at DATETIME2,
    refresh_token_expires_at DATETIME2,
    scope NVARCHAR(MAX),
    id_token NVARCHAR(MAX),
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE(),
    CONSTRAINT UQ_Provider_Account UNIQUE (provider_id, account_id),
    CONSTRAINT FK_Account_User FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='verifications' AND xtype='U')
CREATE TABLE verifications (
    id %s,
    identifier NVARCHAR(255) NOT NULL,
    token NVARCHAR(255) NOT NULL UNIQUE,
    type NVARCHAR(50) NOT NULL,
    expires_at DATETIME2 NOT NULL,
    created_at DATETIME2 DEFAULT GETDATE(),
    updated_at DATETIME2 DEFAULT GETDATE()
);
`, idDef, idDef, fkDef, idDef, fkDef, idDef)
}

func generateMSSQLTwoFA(idType string) string {
	idDef := "NVARCHAR(255) PRIMARY KEY"
	if idType == "uuid" {
		idDef = "UNIQUEIDENTIFIER PRIMARY KEY DEFAULT NEWID()"
	} else if idType == "serial" {
		idDef = "INT IDENTITY(1,1) PRIMARY KEY"
	}

	fkDef := "NVARCHAR(255)"
	if idType == "uuid" {
		fkDef = "UNIQUEIDENTIFIER"
	} else if idType == "serial" {
		fkDef = "INT"
	}

	return fmt.Sprintf(`IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='two_factors' AND xtype='U')
CREATE TABLE two_factors (
    id %s,
    user_id %s NOT NULL,
    secret NVARCHAR(MAX) NOT NULL,
    uri NVARCHAR(MAX) NOT NULL,
    created_at DATETIME2 DEFAULT GETDATE(),
    CONSTRAINT FK_TwoFactor_User FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

IF NOT EXISTS (SELECT * FROM sysobjects WHERE name='two_factor_backup_codes' AND xtype='U')
CREATE TABLE two_factor_backup_codes (
    id %s,
    user_id %s NOT NULL,
    code NVARCHAR(255) NOT NULL,
    created_at DATETIME2 DEFAULT GETDATE(),
    CONSTRAINT FK_BackupCode_User FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
`, idDef, fkDef, idDef, fkDef)
}
