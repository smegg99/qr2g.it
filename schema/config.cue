package schema

@go(config)

#Config: {
	server:      #ServerConfig
	db:          #DBConfig
	github:      #GithubConfig
	turnstile:   #TurnstileConfig
	verifier:    #VerifierConfig
	policy:      #PolicyConfig
	statistics:  #StatisticsConfig
	contact:     #ContactConfig
	ratelimit:   #RateLimitConfig
	logger:      #LoggerConfig
	preferences: #Preferences
}

#TrustedProxyMode: "cloudflare" | "none" | "all"
#GinMode:          "release" | "debug" | "test"

#ServerConfig: {
	listen_addr:                 string | *":8080"                 @go(ListenAddr)
	public_base_url:             string | *"https://qr2g.it"       @go(PublicBaseURL)
	canonical_base_url:          string | *"HTTPS://QR2G.IT"       @go(CanonicalBaseURL)
	trusted_proxy_mode:          #TrustedProxyMode | *"cloudflare" @go(TrustedProxyMode)
	max_body_bytes:              int & >0 | *4096                  @go(MaxBodyBytes)
	gin_mode:                    #GinMode | *"release"             @go(GinMode)
	shutdown_timeout_seconds:    int & >0 | *15                    @go(ShutdownTimeoutSeconds)
	read_header_timeout_seconds: int & >0 | *5                     @go(ReadHeaderTimeoutSeconds)
	write_timeout_seconds:       int & >0 | *15                    @go(WriteTimeoutSeconds)
	idle_timeout_seconds:        int & >0 | *60                    @go(IdleTimeoutSeconds)
	require_shorthand_turnstile: bool | *false                     @go(RequireShorthandTurnstile)
	service_name:                string | *"qr2g.it"               @go(ServiceName)
}

#DBConfig: {
	path:         string & !="" | *"data/qr2git.sqlite" @go(Path)
	auto_migrate: bool | *true                          @go(AutoMigrate)
}

#GithubConfig: {
	user_agent:              string | *"qr2g.it"                @go(UserAgent)
	api_version:             string | *"2026-03-10"             @go(APIVersion)
	rest_base_url:           string | *"https://api.github.com" @go(RESTBaseURL)
	request_timeout_seconds: int & >0 | *15                     @go(RequestTimeoutSeconds)
	// Optional token for public-data reads; anonymous reads have a small hourly allowance.
	token: string | *"@{env?:GITHUB_TOKEN}" @go(Token)
	// Background checks pause while GitHub reports fewer remaining requests than this.
	background_reserve: int & >=0 | *15 @go(BackgroundReserve)
}

#TurnstileConfig: {
	secret_key:              string | *"@{env?:TURNSTILE_SECRET_KEY}"                              @go(SecretKey)
	site_key:                string | *"@{pubenv?:TURNSTILE_SITE_KEY}"                             @go(SiteKey)
	siteverify_url:          string | *"https://challenges.cloudflare.com/turnstile/v0/siteverify" @go(SiteverifyURL)
	expected_hostname:       string | *""                                                          @go(ExpectedHostname)
	enabled:                 bool | *true                                                          @go(Enabled)
	request_timeout_seconds: int & >0 | *5                                                         @go(RequestTimeoutSeconds)
}

#VerifierConfig: {
	enabled:                 bool | *true     @go(Enabled)
	interval_seconds:        int & >0 | *60   @go(IntervalSeconds)
	batch_size:              int & >0 | *50   @go(BatchSize)
	concurrency:             int & >0 | *4    @go(Concurrency)
	fresh_ttl_seconds:       int & >0 | *3600 @go(FreshTTLSeconds)
	stale_jitter_seconds:    int & >=0 | *600 @go(StaleJitterSeconds)
	max_fail_count:          int & >0 | *10   @go(MaxFailCount)
	backoff_base_seconds:    int & >0 | *30   @go(BackoffBaseSeconds)
	backoff_max_seconds:     int & >0 | *3600 @go(BackoffMaxSeconds)
	request_timeout_seconds: int & >0 | *15   @go(RequestTimeoutSeconds)
}

#PolicyConfig: {
	enabled:                  bool | *true      @go(Enabled)
	interval_seconds:         int & >0 | *60    @go(IntervalSeconds)
	batch_size:               int & >0 | *5     @go(BatchSize)
	fresh_ttl_seconds:        int & >0 | *86400 @go(FreshTTLSeconds)
	failure_backoff_seconds:  int & >0 | *1800  @go(FailureBackoffSeconds)
	refresh_cooldown_seconds: int & >0 | *300   @go(RefreshCooldownSeconds)
}

#StatisticsConfig: {
	flush_interval_seconds: int & >0 | *10 @go(FlushIntervalSeconds)
}

#ContactConfig: {
	email:    string | *"@{env?:CONTACT_EMAIL}"          @go(Email)
	catchall: string | *"@{env?:CONTACT_CATCHALL_EMAIL}" @go(Catchall)
}

#RateLimitConfig: {
	create_per_min:     int & >0 | *3   @go(CreatePerMin)
	create_per_day:     int & >0 | *50  @go(CreatePerDay)
	resolve_per_min:    int & >0 | *600 @go(ResolvePerMin)
	miss_per_min:       int & >0 | *60  @go(MissPerMin)
	shorthand_per_hour: int & >0 | *5   @go(ShorthandPerHour)
	asset_per_min:      int & >0 | *600 @go(AssetPerMin)
	read_per_min:       int & >0 | *120 @go(ReadPerMin)
	refresh_per_hour:   int & >0 | *20  @go(RefreshPerHour)
	contact_per_min:    int & >0 | *6   @go(ContactPerMin)
	github_per_sec:     int & >0 | *10  @go(GithubPerSec)
	github_burst:       int & >0 | *20  @go(GithubBurst)
	cleanup_seconds:    int & >0 | *600 @go(CleanupSeconds)
}

#LocaleCode: "en" | "pl"

#Preferences: {
	language: #LocaleCode | *"en"
}

#LoggerConfig: {
	verbose:              bool | *false
	no_color:             bool | *false @go(NoColor)
	show_caller:          bool | *false @go(ShowCaller)
	dir:                  string | *""
	max_size_mb:          int & >0 | *10       @go(MaxSizeMb)
	max_backups:          int & >=0 | *5       @go(MaxBackups)
	max_age_days:         int & >0 | *30       @go(MaxAgeDays)
	log_name:             string | *"qr2g.log" @go(LogName)
	frontend_console_log: bool | *true         @go(FrontendConsoleLog)
}
