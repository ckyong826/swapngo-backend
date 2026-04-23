package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// AppConfig holds all global environment variables
type Config struct {
	SUIChainURL string
	SOLChainURL string
	EVMChainURL string
	JWTSecret      []byte
	JWTAccessTime  time.Duration
	JWTRefreshTime time.Duration
	SUITreasuryAddress string
	SUIPackageID string
	SUITreasuryCapID string
	SUITreasuryPriv string
	SUIAdminPriv string
	SUIAdminAddress string
	KafkaBrokers string
}

// Global instance
var Env *Config

// Load initializes the environment variables
func Load() {
	// Load .env file (it will silently skip if the file doesn't exist, 
	// which is great for Docker/Production where env vars are injected directly)
	if err := godotenv.Load(); err != nil {
		log.Println("Notice: No .env file found, relying on system environment variables")
	}

	// Safely parse Access Time (Fallback to 15m if missing or invalid)
	accessTime, err := time.ParseDuration(os.Getenv("JWT_ACCESS_TIME"))
	if err != nil {
		log.Println("Warning: Invalid JWT_ACCESS_TIME, defaulting to 15m")
		accessTime = 15 * time.Minute
	}

	// Safely parse Refresh Time (Fallback to 7 days if missing)
	refreshTime, err := time.ParseDuration(os.Getenv("JWT_REFRESH_TIME"))
	if err != nil {
		log.Println("Warning: Invalid JWT_REFRESH_TIME, defaulting to 168h")
		refreshTime = 168 * time.Hour
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("FATAL: JWT_SECRET environment variable is required but missing!")
	}

	suiChainURL := os.Getenv("SUI_CHAIN_URL")
	if suiChainURL == "" {
		log.Fatal("FATAL: SUI_CHAIN_URL environment variable is required but missing!")
	}

	solChainURL := os.Getenv("SOL_CHAIN_URL")
	if solChainURL == "" {
		log.Fatal("FATAL: SOL_CHAIN_URL environment variable is required but missing!")
	}

	evmChainURL := os.Getenv("EVM_CHAIN_URL")
	if evmChainURL == "" {
		log.Fatal("FATAL: EVM_CHAIN_URL environment variable is required but missing!")
	}

	suiTreasuryAddress := os.Getenv("SUI_TREASURY_ADDRESS")
	if suiTreasuryAddress == "" {
		log.Fatal("FATAL: SUI_TREASURY_ADDRESS environment variable is required but missing!")
	}

	suiPackageID := os.Getenv("MYRC_SUI_PACKAGE_ID")
	if suiPackageID == "" {
		log.Fatal("FATAL: SUI_PACKAGE_ID environment variable is required but missing!")
	}

	suiTreasuryCapID := os.Getenv("MYRC_SUI_TREASURY_CAP_ID")
	if suiTreasuryCapID == "" {
		log.Fatal("FATAL: SUI_TREASURY_CAP_ID environment variable is required but missing!")
	}

	suiTreasuryPriv := os.Getenv("SUI_TREASURY_PRIVATE_KEY")
	if suiTreasuryPriv == "" {
		log.Fatal("FATAL: SUI_TREASURY_PRIVATE_KEY environment variable is required but missing!")
	}

	suiAdminPriv := os.Getenv("SUI_ADMIN_PRIVATE_KEY")
	if suiAdminPriv == "" {
		log.Fatal("FATAL: SUI_ADMIN_PRIVATE_KEY environment variable is required but missing!")
	}	

	suiAdminAddress := os.Getenv("SUI_ADMIN_ADDRESS")
	if suiAdminAddress == "" {
		log.Fatal("FATAL: SUI_ADMIN_ADDRESS environment variable is required but missing!")
	}	

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}

	// Populate the global config
	Env = &Config{
		JWTSecret:      []byte(secret),
		JWTAccessTime:  accessTime,
		JWTRefreshTime: refreshTime,
		SUIChainURL:    suiChainURL,
		SOLChainURL:    solChainURL,
		EVMChainURL:    evmChainURL,
		SUITreasuryAddress: suiTreasuryAddress,
		SUIPackageID: suiPackageID,
		SUITreasuryCapID: suiTreasuryCapID,
		SUITreasuryPriv: suiTreasuryPriv,
		SUIAdminPriv: suiAdminPriv,
		SUIAdminAddress: suiAdminAddress,
		KafkaBrokers: kafkaBrokers,
	}
}