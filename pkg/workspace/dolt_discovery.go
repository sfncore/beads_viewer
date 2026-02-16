package workspace

// ConfigFromDatabases builds a synthetic workspace Config from a list of Dolt
// database names. Each database becomes a RepoConfig with DoltDatabase set,
// enabling the AggregateLoader to query Dolt SQL directly.
func ConfigFromDatabases(databases []string) *Config {
	repos := make([]RepoConfig, len(databases))
	for i, db := range databases {
		repos[i] = RepoConfig{
			Name:         db,
			Path:         db, // synthetic — not used for Dolt loading
			Prefix:       db + "-",
			DoltDatabase: db,
		}
	}
	return &Config{
		Name:  "dolt-auto",
		Repos: repos,
	}
}
