package configuration

import (
	"flag"
	"log"
	"os"
	"slices"
	"strconv"
)

const apiUrlBasic = "https://api2.isbndb.com"
const apiUrlPremium = "https://api.premium.isbndb.com"
const apiUrlPro = "https://api.pro.isbndb.com"

type Config struct {
	SearchBy                    string
	File                        string
	Provider                    string
	IsbndbSubscriptionType      string
	IsbndbApiKey                string
	IsbndbApiUrl                string
	CallsPerSecond              int
	TimeoutSeconds              int
	DbHost                      string
	DbPort                      string
	DbUsername                  string
	DbPassword                  string
	DbNameBooks                 string
	DbNameProgress              string
	DbConcurrentWriteGoroutines int
	// todo: implement MaxCallsPerDay
}

func Get() Config {
	var config Config

	callsPerSecond, err := strconv.Atoi(os.Getenv("CALLS_PER_SECOND"))
	if err != nil {
		callsPerSecond = 0
	}
	timeoutSeconds, err := strconv.Atoi(os.Getenv("TIMEOUT_SECONDS"))
	if err != nil {
		timeoutSeconds = 0
	}
	dbConcurrentWriteGoroutines, err := strconv.Atoi(os.Getenv("DB_CONCURRENT_WRITE_GOROUTINES"))
	if err != nil {
		dbConcurrentWriteGoroutines = 0
	}

	config.SearchBy = *flag.String("search-by", os.Getenv("SEARCH_BY"), "Title, subject or isbn.")
	config.File = *flag.String("file", os.Getenv("FILE"), "File to read from.")
	config.Provider = *flag.String("provider", os.Getenv("PROVIDER"), "IsbnDB or Google.")
	config.IsbndbSubscriptionType = *flag.String("isbndb-subscription-type", os.Getenv("ISBNDB_SUBSCRIPTION_TYPE"), "Basic, premium or pro. Required if provider is IsbnDB.")
	config.IsbndbApiKey = *flag.String("isbndb-api-key", os.Getenv("ISBNDB_API_KEY"), "IsbnDB API key. Required if provider is IsbnDB.")
	config.CallsPerSecond = *flag.Int("calls-per-second", callsPerSecond, "The max number of calls per second that should be made to the API.")
	config.TimeoutSeconds = *flag.Int("timeout-seconds", timeoutSeconds, "If the API returns a timeout, how many seconds should be waited before trying again.")
	config.DbHost = *flag.String("db-host", os.Getenv("DB_HOST"), "Database host.")
	config.DbPort = *flag.String("db-port", os.Getenv("DB_PORT"), "Database port.")
	config.DbUsername = *flag.String("db-username", os.Getenv("DB_USERNAME"), "Database username.")
	config.DbPassword = *flag.String("db-password", os.Getenv("DB_PASSWORD"), "Database password.")
	config.DbNameBooks = *flag.String("db-name-books", os.Getenv("DB_NAME_BOOKS"), "The name of the database where books are saved.")
	config.DbNameProgress = *flag.String("db-name-progress", os.Getenv("DB_NAME_PROGRESS"), "The name of the database where progress is saved.")
	config.DbConcurrentWriteGoroutines = *flag.Int("db-concurrent-write-goroutines", dbConcurrentWriteGoroutines, "How many goroutines should be used to write to the database. You should be mindful of how many concurrent threads your database can handle.")

	flag.Parse()

	if config.SearchBy == "" {
		config.SearchBy = "title"
	}

	if config.Provider == "" {
		config.Provider = "isbndb"
	}

	if config.CallsPerSecond == 0 {
		config.CallsPerSecond = 1
	}

	if config.DbNameBooks == "" {
		config.DbNameBooks = "book_data_" + config.Provider
	}

	if config.DbNameProgress == "" {
		config.DbNameProgress = "progress_" + config.Provider
	}

	if config.DbConcurrentWriteGoroutines == 0 {
		config.DbConcurrentWriteGoroutines = 1
	}

	validateConfiguration(config)

	isbndbApiUrls := map[string]string{
		"basic":   apiUrlBasic,
		"premium": apiUrlPremium,
		"pro":     apiUrlPro,
	}
	config.IsbndbApiUrl = isbndbApiUrls[config.IsbndbSubscriptionType]

	return config
}

func validateConfiguration(config Config) {
	validSearchByValues := []string{"title", "subject", "isbn"}
	if !slices.Contains(validSearchByValues, config.SearchBy) {
		log.Fatal("Invalid search by value")
	}

	if config.File == "" {
		log.Fatal("File is not set")
	}

	validProviderValues := []string{"isbndb", "google"}
	if !slices.Contains(validProviderValues, config.Provider) {
		log.Fatal("Invalid provider value")
	}

	validIsbndbSubscriptionTypeValues := []string{"basic", "premium", "pro"}
	if config.Provider == "isbndb" && !slices.Contains(validIsbndbSubscriptionTypeValues, config.IsbndbSubscriptionType) {
		log.Fatal("Invalid isbndb subscription type value")
	}

	if config.Provider == "isbndb" && config.IsbndbApiKey == "" {
		log.Fatal("IsbnDB API key is not set")
	}

	if config.CallsPerSecond < 1 {
		log.Fatal("Invalid calls per second value")
	}

	if config.TimeoutSeconds < 0 {
		log.Fatal("Invalid timeout seconds value")
	}

	if config.DbHost == "" {
		log.Fatal("Database host is not set")
	}

	if config.DbPort == "" {
		log.Fatal("Database port is not set")
	}

	if config.DbUsername == "" {
		log.Fatal("Database username is not set")
	}

	if config.DbPassword == "" {
		log.Fatal("Database password is not set")
	}

	if config.DbNameBooks == "" {
		log.Fatal("Database name books is not set")
	}

	if config.DbNameProgress == "" {
		log.Fatal("Database name progress is not set")
	}

	if config.DbConcurrentWriteGoroutines < 1 {
		log.Fatal("Invalid database concurrent write goroutines value")
	}
}
