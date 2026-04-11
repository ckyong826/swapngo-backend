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
	TreasuryAddress string
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

	treasuryAddress := os.Getenv("TREASURY_ADDRESS")
	if treasuryAddress == "" {
		log.Fatal("FATAL: TREASURY_ADDRESS environment variable is required but missing!")
	}

	// Populate the global config
	Env = &Config{
		JWTSecret:      []byte(secret),
		JWTAccessTime:  accessTime,
		JWTRefreshTime: refreshTime,
		SUIChainURL:    suiChainURL,
		SOLChainURL:    solChainURL,
		EVMChainURL:    evmChainURL,
		TreasuryAddress: treasuryAddress,
	}
}