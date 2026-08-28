package request

type Log struct {
	Model         string        `json:"model"`
	StreamOptions StreamOptions `json:"stream_options"`
	Stream        bool          `json:"stream"`
	Metadata      CallMetadata  `json:"metadata"`
	SecretFields  SecretFields  `json:"secret_fields"`
	LiteLLMCallID string        `json:"litellm_call_id"`
	ArrivalTime   float64       `json:"arrival_time"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type AuthMetadata struct {
	User string `json:"user"`
}

type RequesterMetadata struct {
	Headers HTTPHeaders `json:"headers"`
}

type CallMetadata struct {
	Headers                HTTPHeaders        `json:"headers"`
	RequesterMetadata      RequesterMetadata  `json:"requester_metadata"`
	UserAPIKeyHash         string             `json:"user_api_key_hash"`
	UserAPIKeySpend        float64            `json:"user_api_key_spend"`
	UserAPIKeyRequestRoute string             `json:"user_api_key_request_route"`
	UserAPIKeyAuthMetadata AuthMetadata       `json:"user_api_key_auth_metadata"`
	UserAPIKey             string             `json:"user_api_key"`
	LiteLLMAPIVersion      string             `json:"litellm_api_version"`
	UserAPIKeyMetadata     AuthMetadata       `json:"user_api_key_metadata"`
	Endpoint               string             `json:"endpoint"`
	RequesterIPAddress     string             `json:"requester_ip_address"`
	UserAgent              string             `json:"user_agent"`
	QueueTimeSeconds       float64            `json:"queue_time_seconds"`
	ProxyServerRequest     ProxyServerRequest `json:"proxy_server_request"`
}

type HTTPHeaders struct {
	Host                     string `json:"host"`
	UserAgent                string `json:"user-agent"`
	ContentLength            string `json:"content-length"`
	Accept                   string `json:"accept"`
	ContentType              string `json:"content-type"`
	AcceptEncoding           string `json:"accept-encoding"`
	Authorization            string `json:"authorization,omitempty"`
	XStainlessArch           string `json:"x-stainless-arch"`
	XStainlessLang           string `json:"x-stainless-lang"`
	XStainlessOS             string `json:"x-stainless-os"`
	XStainlessPackageVersion string `json:"x-stainless-package-version"`
	XStainlessRetryCount     string `json:"x-stainless-retry-count"`
	XStainlessRuntime        string `json:"x-stainless-runtime"`
	XStainlessRuntimeVersion string `json:"x-stainless-runtime-version"`
}

type ProxyServerRequest struct {
	URL     string      `json:"url"`
	Method  string      `json:"method"`
	Headers HTTPHeaders `json:"headers"`
}

type SecretFields struct {
	RawHeaders HTTPHeaders `json:"raw_headers"`
}
