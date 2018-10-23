// Package alertcenter provides access to the G Suite Alert Center API.
//
// See https://developers.google.com/admin-sdk/alertcenter/
//
// Usage example:
//
//   import "google.golang.org/api/alertcenter/v1beta1"
//   ...
//   alertcenterService, err := alertcenter.New(oauthHttpClient)
package alertcenter // import "google.golang.org/api/alertcenter/v1beta1"

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	context "golang.org/x/net/context"
	ctxhttp "golang.org/x/net/context/ctxhttp"
	gensupport "google.golang.org/api/gensupport"
	googleapi "google.golang.org/api/googleapi"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// Always reference these packages, just in case the auto-generated code
// below doesn't.
var _ = bytes.NewBuffer
var _ = strconv.Itoa
var _ = fmt.Sprintf
var _ = json.NewDecoder
var _ = io.Copy
var _ = url.Parse
var _ = gensupport.MarshalJSON
var _ = googleapi.Version
var _ = errors.New
var _ = strings.Replace
var _ = context.Canceled
var _ = ctxhttp.Do

const apiId = "alertcenter:v1beta1"
const apiName = "alertcenter"
const apiVersion = "v1beta1"
const basePath = "https://alertcenter.googleapis.com/"

func New(client *http.Client) (*Service, error) {
	if client == nil {
		return nil, errors.New("client is nil")
	}
	s := &Service{client: client, BasePath: basePath}
	s.Alerts = NewAlertsService(s)
	return s, nil
}

type Service struct {
	client    *http.Client
	BasePath  string // API endpoint base URL
	UserAgent string // optional additional User-Agent fragment

	Alerts *AlertsService
}

func (s *Service) userAgent() string {
	if s.UserAgent == "" {
		return googleapi.UserAgent
	}
	return googleapi.UserAgent + " " + s.UserAgent
}

func NewAlertsService(s *Service) *AlertsService {
	rs := &AlertsService{s: s}
	rs.Feedback = NewAlertsFeedbackService(s)
	return rs
}

type AlertsService struct {
	s *Service

	Feedback *AlertsFeedbackService
}

func NewAlertsFeedbackService(s *Service) *AlertsFeedbackService {
	rs := &AlertsFeedbackService{s: s}
	return rs
}

type AlertsFeedbackService struct {
	s *Service
}

// AccountWarning: Alerts for user account warning events.
type AccountWarning struct {
	// Email: Required. Email of the user that this event belongs to.
	Email string `json:"email,omitempty"`

	// LoginDetails: Optional. Details of the login action associated with
	// the warning event.
	// This is only available for:
	// Suspicious login
	// Suspicious login (less secure app)
	// User suspended (suspicious activity)
	LoginDetails *LoginDetails `json:"loginDetails,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Email") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Email") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *AccountWarning) MarshalJSON() ([]byte, error) {
	type NoMethod AccountWarning
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// Alert: An alert affecting a customer.
// All fields are read-only once created.
type Alert struct {
	// AlertId: Output only. The unique identifier for the alert.
	AlertId string `json:"alertId,omitempty"`

	// CreateTime: Output only. The time this alert was created. Assigned by
	// the server.
	CreateTime string `json:"createTime,omitempty"`

	// CustomerId: Output only. The unique identifier of the Google account
	// of the customer.
	CustomerId string `json:"customerId,omitempty"`

	// Data: Optional. Specific data associated with this alert.
	// e.g. google.apps.alertcenter.type.DeviceCompromised.
	Data googleapi.RawMessage `json:"data,omitempty"`

	// Deleted: Output only. Whether this alert has been marked for
	// deletion.
	Deleted bool `json:"deleted,omitempty"`

	// EndTime: Optional. The time this alert was no longer active. If
	// provided, the
	// end time must not be earlier than the start time. If not provided,
	// the end
	// time will default to the start time.
	EndTime string `json:"endTime,omitempty"`

	// SecurityInvestigationToolLink: Output only. An optional Security
	// Investigation Tool query for this
	// alert.
	SecurityInvestigationToolLink string `json:"securityInvestigationToolLink,omitempty"`

	// Source: Required. A unique identifier for the system that is reported
	// the alert.
	//
	// Supported sources are any of the following:
	//  * "Google Operations"
	//  * "Mobile device management"
	//  * "Gmail phishing"
	//  * "Domain wide takeout"
	//  * "Government attack warning"
	//  * "Google identity"
	Source string `json:"source,omitempty"`

	// StartTime: Required. The time this alert became active.
	StartTime string `json:"startTime,omitempty"`

	// Type: Required. The type of the alert.
	//
	// Supported types are any of the following:
	//  * "Google Operations"
	//  * "Device compromised"
	//  * "Suspicious activity"
	//  * "User reported phishing"
	//  * "Misconfigured whitelist"
	//  * "Customer takeout initiated"
	//  * "Government attack warning"
	//  * "User reported spam spike"
	//  * "Suspicious message reported"
	//  * "Phishing reclassification"
	//  * "Malware reclassification"
	// LINT.IfChange
	//  * "Suspicious login"
	//  * "Suspicious login (less secure app)"
	//  * "User suspended"
	//  * "Leaked password"
	//  * "User suspended (suspicious activity)"
	//  * "User suspended (spam)"
	//  * "User suspended (spam through
	// relay)"
	// LINT.ThenChange(//depot/google3/apps/albert/data/albert_enums.
	// proto)
	Type string `json:"type,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "AlertId") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "AlertId") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *Alert) MarshalJSON() ([]byte, error) {
	type NoMethod Alert
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// AlertFeedback: A customer feedback about an alert.
type AlertFeedback struct {
	// AlertId: Output only. The alert identifier.
	AlertId string `json:"alertId,omitempty"`

	// CreateTime: Output only. The time this feedback was created. Assigned
	// by the server.
	CreateTime string `json:"createTime,omitempty"`

	// CustomerId: Output only. The unique identifier of the Google account
	// of the customer.
	CustomerId string `json:"customerId,omitempty"`

	// Email: Output only. The email of the user that provided the feedback.
	Email string `json:"email,omitempty"`

	// FeedbackId: Output only. A unique identifier for the feedback. When
	// creating a new
	// feedback the system will assign one.
	FeedbackId string `json:"feedbackId,omitempty"`

	// Type: Required. The type of the feedback.
	//
	// Possible values:
	//   "ALERT_FEEDBACK_TYPE_UNSPECIFIED" - Feedback type is not specified.
	//   "NOT_USEFUL" - Alert report is not useful.
	//   "SOMEWHAT_USEFUL" - Alert report is somewhat useful.
	//   "VERY_USEFUL" - Alert report is very useful.
	Type string `json:"type,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "AlertId") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "AlertId") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *AlertFeedback) MarshalJSON() ([]byte, error) {
	type NoMethod AlertFeedback
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// Attachment: Attachment with application-specific information about an
// alert.
type Attachment struct {
	// Csv: CSV file attachment.
	Csv *Csv `json:"csv,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Csv") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Csv") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *Attachment) MarshalJSON() ([]byte, error) {
	type NoMethod Attachment
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// BadWhitelist: Alert for setting the domain or ip that malicious email
// comes from as
// whitelisted domain or ip in Gmail advanced settings.
type BadWhitelist struct {
	// DomainId: Domain id.
	DomainId *DomainId `json:"domainId,omitempty"`

	// MaliciousEntity: Entity whose actions triggered a Gmail phishing
	// alert.
	MaliciousEntity *MaliciousEntity `json:"maliciousEntity,omitempty"`

	// Messages: Every alert could contain multiple messages.
	Messages []*GmailMessageInfo `json:"messages,omitempty"`

	// SourceIp: The source ip address of the malicious email. e.g.
	// "127.0.0.1".
	SourceIp string `json:"sourceIp,omitempty"`

	// ForceSendFields is a list of field names (e.g. "DomainId") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "DomainId") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *BadWhitelist) MarshalJSON() ([]byte, error) {
	type NoMethod BadWhitelist
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// Csv: Representation of a CSV file attachment, as a list of column
// headers and
// a list of data rows.
type Csv struct {
	// DataRows: List of data rows in a CSV file, as string arrays rather
	// than as a
	// single comma-separated string.
	DataRows []*CsvRow `json:"dataRows,omitempty"`

	// Headers: List of headers for data columns in a CSV file.
	Headers []string `json:"headers,omitempty"`

	// ForceSendFields is a list of field names (e.g. "DataRows") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "DataRows") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *Csv) MarshalJSON() ([]byte, error) {
	type NoMethod Csv
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// CsvRow: Representation of a single data row in a CSV file.
type CsvRow struct {
	// Entries: Data entries in a CSV file row, as a string array rather
	// than a single
	// comma-separated string.
	Entries []string `json:"entries,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Entries") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Entries") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *CsvRow) MarshalJSON() ([]byte, error) {
	type NoMethod CsvRow
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// DeviceCompromised: A mobile device compromised alert. Derived from
// audit logs.
type DeviceCompromised struct {
	// Email: The email of the user this alert was created for.
	Email string `json:"email,omitempty"`

	// Events: Required. List of security events.
	Events []*DeviceCompromisedSecurityDetail `json:"events,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Email") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Email") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *DeviceCompromised) MarshalJSON() ([]byte, error) {
	type NoMethod DeviceCompromised
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// DeviceCompromisedSecurityDetail: Detailed information of a single MDM
// device compromised event.
type DeviceCompromisedSecurityDetail struct {
	// DeviceCompromisedState: Device compromised state includes:
	// "Compromised" and "Not Compromised".
	DeviceCompromisedState string `json:"deviceCompromisedState,omitempty"`

	// DeviceId: Required. Device Info.
	DeviceId string `json:"deviceId,omitempty"`

	// DeviceModel: The model of the device.
	DeviceModel string `json:"deviceModel,omitempty"`

	// DeviceType: The type of the device.
	DeviceType string `json:"deviceType,omitempty"`

	// IosVendorId: Required for IOS, empty for others.
	IosVendorId string `json:"iosVendorId,omitempty"`

	// ResourceId: The device resource id.
	ResourceId string `json:"resourceId,omitempty"`

	// SerialNumber: The serial number of the device.
	SerialNumber string `json:"serialNumber,omitempty"`

	// ForceSendFields is a list of field names (e.g.
	// "DeviceCompromisedState") to unconditionally include in API requests.
	// By default, fields with empty values are omitted from API requests.
	// However, any non-pointer, non-interface field appearing in
	// ForceSendFields will be sent to the server regardless of whether the
	// field is empty or not. This may be used to include empty fields in
	// Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "DeviceCompromisedState")
	// to include in API requests with the JSON null value. By default,
	// fields with empty values are omitted from API requests. However, any
	// field with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *DeviceCompromisedSecurityDetail) MarshalJSON() ([]byte, error) {
	type NoMethod DeviceCompromisedSecurityDetail
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// DomainId: Domain id of Gmail phishing alerts.
type DomainId struct {
	// CustomerPrimaryDomain: The primary domain for the customer.
	CustomerPrimaryDomain string `json:"customerPrimaryDomain,omitempty"`

	// ForceSendFields is a list of field names (e.g.
	// "CustomerPrimaryDomain") to unconditionally include in API requests.
	// By default, fields with empty values are omitted from API requests.
	// However, any non-pointer, non-interface field appearing in
	// ForceSendFields will be sent to the server regardless of whether the
	// field is empty or not. This may be used to include empty fields in
	// Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "CustomerPrimaryDomain") to
	// include in API requests with the JSON null value. By default, fields
	// with empty values are omitted from API requests. However, any field
	// with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *DomainId) MarshalJSON() ([]byte, error) {
	type NoMethod DomainId
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// DomainWideTakeoutInitiated: A takeout operation for the entire domain
// was initiated by an admin. Derived
// from audit logs.
type DomainWideTakeoutInitiated struct {
	// Email: Email of the admin who initiated the takeout.
	Email string `json:"email,omitempty"`

	// TakeoutRequestId: takeout request id.
	TakeoutRequestId string `json:"takeoutRequestId,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Email") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Email") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *DomainWideTakeoutInitiated) MarshalJSON() ([]byte, error) {
	type NoMethod DomainWideTakeoutInitiated
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// Empty: A generic empty message that you can re-use to avoid defining
// duplicated
// empty messages in your APIs. A typical example is to use it as the
// request
// or the response type of an API method. For instance:
//
//     service Foo {
//       rpc Bar(google.protobuf.Empty) returns
// (google.protobuf.Empty);
//     }
//
// The JSON representation for `Empty` is empty JSON object `{}`.
type Empty struct {
	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`
}

// GmailMessageInfo: Details of a message in phishing spike alert.
type GmailMessageInfo struct {
	// AttachmentsSha256Hash: SHA256 Hash of email's attachment and all MIME
	// parts.
	AttachmentsSha256Hash []string `json:"attachmentsSha256Hash,omitempty"`

	// Date: The date the malicious email was sent.
	Date string `json:"date,omitempty"`

	// Md5HashMessageBody: Hash of message body text.
	Md5HashMessageBody string `json:"md5HashMessageBody,omitempty"`

	// Md5HashSubject: MD5 Hash of email's subject. (Only available for
	// reported emails).
	Md5HashSubject string `json:"md5HashSubject,omitempty"`

	// MessageBodySnippet: Snippet of the message body text. (Only available
	// for reported emails)
	MessageBodySnippet string `json:"messageBodySnippet,omitempty"`

	// MessageId: Message id.
	MessageId string `json:"messageId,omitempty"`

	// Recipient: Recipient of this email.
	Recipient string `json:"recipient,omitempty"`

	// SubjectText: Email subject text. (Only available for reported
	// emails).
	SubjectText string `json:"subjectText,omitempty"`

	// ForceSendFields is a list of field names (e.g.
	// "AttachmentsSha256Hash") to unconditionally include in API requests.
	// By default, fields with empty values are omitted from API requests.
	// However, any non-pointer, non-interface field appearing in
	// ForceSendFields will be sent to the server regardless of whether the
	// field is empty or not. This may be used to include empty fields in
	// Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "AttachmentsSha256Hash") to
	// include in API requests with the JSON null value. By default, fields
	// with empty values are omitted from API requests. However, any field
	// with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *GmailMessageInfo) MarshalJSON() ([]byte, error) {
	type NoMethod GmailMessageInfo
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// GoogleOperations: An incident reported by Google Operations for a G
// Suite application.
type GoogleOperations struct {
	// AffectedUserEmails: List of emails which correspond to the users
	// directly affected by the
	// incident.
	AffectedUserEmails []string `json:"affectedUserEmails,omitempty"`

	// AttachmentData: Optional application-specific data for an incident,
	// provided when the
	// G Suite application which reported the incident cannot be
	// completely
	// restored to a valid state.
	AttachmentData *Attachment `json:"attachmentData,omitempty"`

	// Description: Detailed, freeform incident description.
	Description string `json:"description,omitempty"`

	// Title: One-line incident description.
	Title string `json:"title,omitempty"`

	// ForceSendFields is a list of field names (e.g. "AffectedUserEmails")
	// to unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "AffectedUserEmails") to
	// include in API requests with the JSON null value. By default, fields
	// with empty values are omitted from API requests. However, any field
	// with an empty value appearing in NullFields will be sent to the
	// server as null. It is an error if a field in this list has a
	// non-empty value. This may be used to include null fields in Patch
	// requests.
	NullFields []string `json:"-"`
}

func (s *GoogleOperations) MarshalJSON() ([]byte, error) {
	type NoMethod GoogleOperations
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// ListAlertFeedbackResponse: Response message for an alert feedback
// listing request.
type ListAlertFeedbackResponse struct {
	// Feedback: The list of alert feedback.
	// Result is ordered descending by creation time.
	Feedback []*AlertFeedback `json:"feedback,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "Feedback") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Feedback") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *ListAlertFeedbackResponse) MarshalJSON() ([]byte, error) {
	type NoMethod ListAlertFeedbackResponse
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// ListAlertsResponse: Response message for an alert listing request.
type ListAlertsResponse struct {
	// Alerts: The list of alerts.
	Alerts []*Alert `json:"alerts,omitempty"`

	// NextPageToken: If not empty, indicates that there may be more alerts
	// that match the
	// request; this value can be passed in a new ListAlertsRequest to get
	// the
	// next page of values.
	NextPageToken string `json:"nextPageToken,omitempty"`

	// ServerResponse contains the HTTP response code and headers from the
	// server.
	googleapi.ServerResponse `json:"-"`

	// ForceSendFields is a list of field names (e.g. "Alerts") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Alerts") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *ListAlertsResponse) MarshalJSON() ([]byte, error) {
	type NoMethod ListAlertsResponse
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// LoginDetails: Details of the login action
type LoginDetails struct {
	// IpAddress: Optional. Human readable IP address (e.g., 11.22.33.44)
	// that is
	// associated with the warning event.
	IpAddress string `json:"ipAddress,omitempty"`

	// LoginTime: Optional. Login time that is associated with the warning
	// event.
	LoginTime string `json:"loginTime,omitempty"`

	// ForceSendFields is a list of field names (e.g. "IpAddress") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "IpAddress") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *LoginDetails) MarshalJSON() ([]byte, error) {
	type NoMethod LoginDetails
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// MailPhishing: Proto for all phishing alerts with common
// payload.
// Supported types are any of the following:
// User reported phishing
// User reported spam spike
// Suspicious message reported
// Phishing reclassification
// Malware reclassification
type MailPhishing struct {
	// DomainId: Domain id.
	DomainId *DomainId `json:"domainId,omitempty"`

	// IsInternal: If true, the email is originated from within the
	// organization.
	IsInternal bool `json:"isInternal,omitempty"`

	// MaliciousEntity: Entity whose actions triggered a Gmail phishing
	// alert.
	MaliciousEntity *MaliciousEntity `json:"maliciousEntity,omitempty"`

	// Messages: Every alert could contain multiple messages.
	Messages []*GmailMessageInfo `json:"messages,omitempty"`

	// ForceSendFields is a list of field names (e.g. "DomainId") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "DomainId") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *MailPhishing) MarshalJSON() ([]byte, error) {
	type NoMethod MailPhishing
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// MaliciousEntity: Entity whose actions triggered a Gmail phishing
// alert.
type MaliciousEntity struct {
	// FromHeader: Sender email address.
	FromHeader string `json:"fromHeader,omitempty"`

	// ForceSendFields is a list of field names (e.g. "FromHeader") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "FromHeader") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *MaliciousEntity) MarshalJSON() ([]byte, error) {
	type NoMethod MaliciousEntity
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// PhishingSpike: Alert for a spike in user reported phishing.
// This will be deprecated in favor of MailPhishing.
type PhishingSpike struct {
	// DomainId: Domain id.
	DomainId *DomainId `json:"domainId,omitempty"`

	// IsInternal: If true, the email is originated from within the
	// organization.
	IsInternal bool `json:"isInternal,omitempty"`

	// MaliciousEntity: Entity whose actions triggered a Gmail phishing
	// alert.
	MaliciousEntity *MaliciousEntity `json:"maliciousEntity,omitempty"`

	// Messages: Every alert could contain multiple messages.
	Messages []*GmailMessageInfo `json:"messages,omitempty"`

	// ForceSendFields is a list of field names (e.g. "DomainId") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "DomainId") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *PhishingSpike) MarshalJSON() ([]byte, error) {
	type NoMethod PhishingSpike
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// StateSponsoredAttack: A state sponsored attack alert. Derived from
// audit logs.
type StateSponsoredAttack struct {
	// Email: The email of the user this incident was created for.
	Email string `json:"email,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Email") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Email") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *StateSponsoredAttack) MarshalJSON() ([]byte, error) {
	type NoMethod StateSponsoredAttack
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// SuspiciousActivity: A mobile suspicious activity alert. Derived from
// audit logs.
type SuspiciousActivity struct {
	// Email: The email of the user this alert was created for.
	Email string `json:"email,omitempty"`

	// Events: Required. List of security events.
	Events []*SuspiciousActivitySecurityDetail `json:"events,omitempty"`

	// ForceSendFields is a list of field names (e.g. "Email") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "Email") to include in API
	// requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *SuspiciousActivity) MarshalJSON() ([]byte, error) {
	type NoMethod SuspiciousActivity
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// SuspiciousActivitySecurityDetail: Detailed information of a single
// MDM suspicious activity event.
type SuspiciousActivitySecurityDetail struct {
	// DeviceId: Required. Device Info.
	DeviceId string `json:"deviceId,omitempty"`

	// DeviceModel: The model of the device.
	DeviceModel string `json:"deviceModel,omitempty"`

	// DeviceProperty: Device property which is changed.
	DeviceProperty string `json:"deviceProperty,omitempty"`

	// DeviceType: The type of the device.
	DeviceType string `json:"deviceType,omitempty"`

	// IosVendorId: Required for IOS, empty for others.
	IosVendorId string `json:"iosVendorId,omitempty"`

	// NewValue: New value of the device property after change.
	NewValue string `json:"newValue,omitempty"`

	// OldValue: Old value of the device property before change.
	OldValue string `json:"oldValue,omitempty"`

	// ResourceId: The device resource id.
	ResourceId string `json:"resourceId,omitempty"`

	// SerialNumber: The serial number of the device.
	SerialNumber string `json:"serialNumber,omitempty"`

	// ForceSendFields is a list of field names (e.g. "DeviceId") to
	// unconditionally include in API requests. By default, fields with
	// empty values are omitted from API requests. However, any non-pointer,
	// non-interface field appearing in ForceSendFields will be sent to the
	// server regardless of whether the field is empty or not. This may be
	// used to include empty fields in Patch requests.
	ForceSendFields []string `json:"-"`

	// NullFields is a list of field names (e.g. "DeviceId") to include in
	// API requests with the JSON null value. By default, fields with empty
	// values are omitted from API requests. However, any field with an
	// empty value appearing in NullFields will be sent to the server as
	// null. It is an error if a field in this list has a non-empty value.
	// This may be used to include null fields in Patch requests.
	NullFields []string `json:"-"`
}

func (s *SuspiciousActivitySecurityDetail) MarshalJSON() ([]byte, error) {
	type NoMethod SuspiciousActivitySecurityDetail
	raw := NoMethod(*s)
	return gensupport.MarshalJSON(raw, s.ForceSendFields, s.NullFields)
}

// method id "alertcenter.alerts.delete":

type AlertsDeleteCall struct {
	s          *Service
	alertId    string
	urlParams_ gensupport.URLParams
	ctx_       context.Context
	header_    http.Header
}

// Delete: Marks the specified alert for deletion. An alert that has
// been marked for
// deletion will be excluded from the results of a List operation by
// default,
// and will be removed from the Alert Center after 30 days.
// Marking an alert for deletion will have no effect on an alert which
// has
// already been marked for deletion. Attempting to mark a nonexistent
// alert
// for deletion will return NOT_FOUND.
func (r *AlertsService) Delete(alertId string) *AlertsDeleteCall {
	c := &AlertsDeleteCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.alertId = alertId
	return c
}

// CustomerId sets the optional parameter "customerId": The unique
// identifier of the Google account of the customer the
// alert is associated with. This is obfuscated and not the plain
// customer
// ID as stored internally. Inferred from the caller identity if not
// provided.
func (c *AlertsDeleteCall) CustomerId(customerId string) *AlertsDeleteCall {
	c.urlParams_.Set("customerId", customerId)
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AlertsDeleteCall) Fields(s ...googleapi.Field) *AlertsDeleteCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *AlertsDeleteCall) Context(ctx context.Context) *AlertsDeleteCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *AlertsDeleteCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *AlertsDeleteCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v1beta1/alerts/{alertId}")
	urls += "?" + c.urlParams_.Encode()
	req, err := http.NewRequest("DELETE", urls, body)
	if err != nil {
		return nil, err
	}
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"alertId": c.alertId,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "alertcenter.alerts.delete" call.
// Exactly one of *Empty or error will be non-nil. Any non-2xx status
// code is an error. Response headers are in either
// *Empty.ServerResponse.Header or (if a response was returned at all)
// in error.(*googleapi.Error).Header. Use googleapi.IsNotModified to
// check whether the returned error was because http.StatusNotModified
// was returned.
func (c *AlertsDeleteCall) Do(opts ...googleapi.CallOption) (*Empty, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Empty{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Marks the specified alert for deletion. An alert that has been marked for\ndeletion will be excluded from the results of a List operation by default,\nand will be removed from the Alert Center after 30 days.\nMarking an alert for deletion will have no effect on an alert which has\nalready been marked for deletion. Attempting to mark a nonexistent alert\nfor deletion will return NOT_FOUND.",
	//   "flatPath": "v1beta1/alerts/{alertId}",
	//   "httpMethod": "DELETE",
	//   "id": "alertcenter.alerts.delete",
	//   "parameterOrder": [
	//     "alertId"
	//   ],
	//   "parameters": {
	//     "alertId": {
	//       "description": "Required. The identifier of the alert to delete.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "customerId": {
	//       "description": "Optional. The unique identifier of the Google account of the customer the\nalert is associated with. This is obfuscated and not the plain customer\nID as stored internally. Inferred from the caller identity if not provided.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "v1beta1/alerts/{alertId}",
	//   "response": {
	//     "$ref": "Empty"
	//   }
	// }

}

// method id "alertcenter.alerts.get":

type AlertsGetCall struct {
	s            *Service
	alertId      string
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// Get: Gets the specified alert.
func (r *AlertsService) Get(alertId string) *AlertsGetCall {
	c := &AlertsGetCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.alertId = alertId
	return c
}

// CustomerId sets the optional parameter "customerId": The unique
// identifier of the Google account of the customer the
// alert is associated with. This is obfuscated and not the plain
// customer
// ID as stored internally. Inferred from the caller identity if not
// provided.
func (c *AlertsGetCall) CustomerId(customerId string) *AlertsGetCall {
	c.urlParams_.Set("customerId", customerId)
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AlertsGetCall) Fields(s ...googleapi.Field) *AlertsGetCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *AlertsGetCall) IfNoneMatch(entityTag string) *AlertsGetCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *AlertsGetCall) Context(ctx context.Context) *AlertsGetCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *AlertsGetCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *AlertsGetCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v1beta1/alerts/{alertId}")
	urls += "?" + c.urlParams_.Encode()
	req, err := http.NewRequest("GET", urls, body)
	if err != nil {
		return nil, err
	}
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"alertId": c.alertId,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "alertcenter.alerts.get" call.
// Exactly one of *Alert or error will be non-nil. Any non-2xx status
// code is an error. Response headers are in either
// *Alert.ServerResponse.Header or (if a response was returned at all)
// in error.(*googleapi.Error).Header. Use googleapi.IsNotModified to
// check whether the returned error was because http.StatusNotModified
// was returned.
func (c *AlertsGetCall) Do(opts ...googleapi.CallOption) (*Alert, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &Alert{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Gets the specified alert.",
	//   "flatPath": "v1beta1/alerts/{alertId}",
	//   "httpMethod": "GET",
	//   "id": "alertcenter.alerts.get",
	//   "parameterOrder": [
	//     "alertId"
	//   ],
	//   "parameters": {
	//     "alertId": {
	//       "description": "Required. The identifier of the alert to retrieve.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "customerId": {
	//       "description": "Optional. The unique identifier of the Google account of the customer the\nalert is associated with. This is obfuscated and not the plain customer\nID as stored internally. Inferred from the caller identity if not provided.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "v1beta1/alerts/{alertId}",
	//   "response": {
	//     "$ref": "Alert"
	//   }
	// }

}

// method id "alertcenter.alerts.list":

type AlertsListCall struct {
	s            *Service
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// List: Lists all the alerts for the current user and application.
func (r *AlertsService) List() *AlertsListCall {
	c := &AlertsListCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	return c
}

// CustomerId sets the optional parameter "customerId": The unique
// identifier of the Google account of the customer the
// alerts are associated with. This is obfuscated and not the
// plain
// customer ID as stored internally. Inferred from the caller identity
// if not
// provided.
func (c *AlertsListCall) CustomerId(customerId string) *AlertsListCall {
	c.urlParams_.Set("customerId", customerId)
	return c
}

// Filter sets the optional parameter "filter": Query string for
// filtering alert results.
// This string must be specified as an expression or list of
// expressions,
// using the following grammar:
//
// ### Expressions
//
// An expression has the general form `<field> <operator> <value>`.
//
// A field or value which contains a space or a colon must be enclosed
// by
// double quotes.
//
// #### Operators
//
// Operators follow the BNF specification:
//
// `<equalityOperator> ::= "=" | ":"
//
// `<relationalOperator> ::= "<" | ">" | "<=" | ">="
//
// Relational operators are defined only for timestamp fields.
// Equality
// operators are defined only for string fields.
//
// #### Timestamp fields
//
// The value supplied for a timestamp field must be an
// [RFC 3339](https://tools.ietf.org/html/rfc3339) date-time
// string.
//
// Supported timestamp fields are `create_time`, `start_time`, and
// `end_time`.
//
// #### String fields
//
// The value supplied for a string field may be an arbitrary
// string.
//
// #### Examples
//
// To query for all alerts created on or after April 5,
// 2018:
//
// `create_time >= "2018-04-05T00:00:00Z"
//
// To query for all alerts from the source "Gmail
// phishing":
//
// `source:"Gmail phishing"
//
// ### Joining expressions
//
// Expressions may be joined to form a more complex query. The
// BNF
// specification is:
//
// `<expressionList> ::= <expression> | <expressionList>
// <conjunction>
// <expressionList> | <negation> <expressionList>`
// `<conjunction> ::= "AND" | "OR" | ""
// `<negation> ::= "NOT"
//
// Using the empty string as a conjunction acts as an implicit AND.
//
// The precedence of joining operations, from highest to lowest, is NOT,
// AND,
// OR.
//
// #### Examples
//
// To query for all alerts which started in 2017:
//
// `start_time >= "2017-01-01T00:00:00Z" AND start_time
// <
// "2018-01-01T00:00:00Z"
//
// To query for all user reported phishing alerts from the source
// "Gmail phishing":
//
// `type:"User reported phishing" source:"Gmail phishing"
func (c *AlertsListCall) Filter(filter string) *AlertsListCall {
	c.urlParams_.Set("filter", filter)
	return c
}

// OrderBy sets the optional parameter "orderBy": Sort the list results
// by a certain order.
// If not specified results may be returned in arbitrary order.
// You can sort the results in a descending order based on the
// creation
// timestamp using order_by="create_time desc".
// Currently, only sorting by create_time desc is supported.
func (c *AlertsListCall) OrderBy(orderBy string) *AlertsListCall {
	c.urlParams_.Set("orderBy", orderBy)
	return c
}

// PageSize sets the optional parameter "pageSize": Requested page size.
// Server may return fewer items than
// requested. If unspecified, server will pick an appropriate default.
func (c *AlertsListCall) PageSize(pageSize int64) *AlertsListCall {
	c.urlParams_.Set("pageSize", fmt.Sprint(pageSize))
	return c
}

// PageToken sets the optional parameter "pageToken": A token
// identifying a page of results the server should return.
// If empty, a new iteration is started. To continue an iteration, pass
// in
// the value from the previous ListAlertsResponse's next_page_token
// field.
func (c *AlertsListCall) PageToken(pageToken string) *AlertsListCall {
	c.urlParams_.Set("pageToken", pageToken)
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AlertsListCall) Fields(s ...googleapi.Field) *AlertsListCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *AlertsListCall) IfNoneMatch(entityTag string) *AlertsListCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *AlertsListCall) Context(ctx context.Context) *AlertsListCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *AlertsListCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *AlertsListCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v1beta1/alerts")
	urls += "?" + c.urlParams_.Encode()
	req, err := http.NewRequest("GET", urls, body)
	if err != nil {
		return nil, err
	}
	req.Header = reqHeaders
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "alertcenter.alerts.list" call.
// Exactly one of *ListAlertsResponse or error will be non-nil. Any
// non-2xx status code is an error. Response headers are in either
// *ListAlertsResponse.ServerResponse.Header or (if a response was
// returned at all) in error.(*googleapi.Error).Header. Use
// googleapi.IsNotModified to check whether the returned error was
// because http.StatusNotModified was returned.
func (c *AlertsListCall) Do(opts ...googleapi.CallOption) (*ListAlertsResponse, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &ListAlertsResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all the alerts for the current user and application.",
	//   "flatPath": "v1beta1/alerts",
	//   "httpMethod": "GET",
	//   "id": "alertcenter.alerts.list",
	//   "parameterOrder": [],
	//   "parameters": {
	//     "customerId": {
	//       "description": "Optional. The unique identifier of the Google account of the customer the\nalerts are associated with. This is obfuscated and not the plain\ncustomer ID as stored internally. Inferred from the caller identity if not\nprovided.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "filter": {
	//       "description": "Optional. Query string for filtering alert results.\nThis string must be specified as an expression or list of expressions,\nusing the following grammar:\n\n### Expressions\n\nAn expression has the general form `\u003cfield\u003e \u003coperator\u003e \u003cvalue\u003e`.\n\nA field or value which contains a space or a colon must be enclosed by\ndouble quotes.\n\n#### Operators\n\nOperators follow the BNF specification:\n\n`\u003cequalityOperator\u003e ::= \"=\" | \":\"`\n\n`\u003crelationalOperator\u003e ::= \"\u003c\" | \"\u003e\" | \"\u003c=\" | \"\u003e=\"`\n\nRelational operators are defined only for timestamp fields. Equality\noperators are defined only for string fields.\n\n#### Timestamp fields\n\nThe value supplied for a timestamp field must be an\n[RFC 3339](https://tools.ietf.org/html/rfc3339) date-time string.\n\nSupported timestamp fields are `create_time`, `start_time`, and `end_time`.\n\n#### String fields\n\nThe value supplied for a string field may be an arbitrary string.\n\n#### Examples\n\nTo query for all alerts created on or after April 5, 2018:\n\n`create_time \u003e= \"2018-04-05T00:00:00Z\"`\n\nTo query for all alerts from the source \"Gmail phishing\":\n\n`source:\"Gmail phishing\"`\n\n### Joining expressions\n\nExpressions may be joined to form a more complex query. The BNF\nspecification is:\n\n`\u003cexpressionList\u003e ::= \u003cexpression\u003e | \u003cexpressionList\u003e \u003cconjunction\u003e\n\u003cexpressionList\u003e | \u003cnegation\u003e \u003cexpressionList\u003e`\n`\u003cconjunction\u003e ::= \"AND\" | \"OR\" | \"\"`\n`\u003cnegation\u003e ::= \"NOT\"`\n\nUsing the empty string as a conjunction acts as an implicit AND.\n\nThe precedence of joining operations, from highest to lowest, is NOT, AND,\nOR.\n\n#### Examples\n\nTo query for all alerts which started in 2017:\n\n`start_time \u003e= \"2017-01-01T00:00:00Z\" AND start_time \u003c\n\"2018-01-01T00:00:00Z\"`\n\nTo query for all user reported phishing alerts from the source\n\"Gmail phishing\":\n\n`type:\"User reported phishing\" source:\"Gmail phishing\"`",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "orderBy": {
	//       "description": "Optional. Sort the list results by a certain order.\nIf not specified results may be returned in arbitrary order.\nYou can sort the results in a descending order based on the creation\ntimestamp using order_by=\"create_time desc\".\nCurrently, only sorting by create_time desc is supported.",
	//       "location": "query",
	//       "type": "string"
	//     },
	//     "pageSize": {
	//       "description": "Optional. Requested page size. Server may return fewer items than\nrequested. If unspecified, server will pick an appropriate default.",
	//       "format": "int32",
	//       "location": "query",
	//       "type": "integer"
	//     },
	//     "pageToken": {
	//       "description": "Optional. A token identifying a page of results the server should return.\nIf empty, a new iteration is started. To continue an iteration, pass in\nthe value from the previous ListAlertsResponse's next_page_token field.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "v1beta1/alerts",
	//   "response": {
	//     "$ref": "ListAlertsResponse"
	//   }
	// }

}

// Pages invokes f for each page of results.
// A non-nil error returned from f will halt the iteration.
// The provided context supersedes any context provided to the Context method.
func (c *AlertsListCall) Pages(ctx context.Context, f func(*ListAlertsResponse) error) error {
	c.ctx_ = ctx
	defer c.PageToken(c.urlParams_.Get("pageToken")) // reset paging to original point
	for {
		x, err := c.Do()
		if err != nil {
			return err
		}
		if err := f(x); err != nil {
			return err
		}
		if x.NextPageToken == "" {
			return nil
		}
		c.PageToken(x.NextPageToken)
	}
}

// method id "alertcenter.alerts.feedback.create":

type AlertsFeedbackCreateCall struct {
	s             *Service
	alertId       string
	alertfeedback *AlertFeedback
	urlParams_    gensupport.URLParams
	ctx_          context.Context
	header_       http.Header
}

// Create: Creates a new alert feedback.
func (r *AlertsFeedbackService) Create(alertId string, alertfeedback *AlertFeedback) *AlertsFeedbackCreateCall {
	c := &AlertsFeedbackCreateCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.alertId = alertId
	c.alertfeedback = alertfeedback
	return c
}

// CustomerId sets the optional parameter "customerId": The unique
// identifier of the Google account of the customer the
// alert's feedback is associated with. This is obfuscated and not
// the
// plain customer ID as stored internally. Inferred from the caller
// identity
// if not provided.
func (c *AlertsFeedbackCreateCall) CustomerId(customerId string) *AlertsFeedbackCreateCall {
	c.urlParams_.Set("customerId", customerId)
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AlertsFeedbackCreateCall) Fields(s ...googleapi.Field) *AlertsFeedbackCreateCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *AlertsFeedbackCreateCall) Context(ctx context.Context) *AlertsFeedbackCreateCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *AlertsFeedbackCreateCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *AlertsFeedbackCreateCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	var body io.Reader = nil
	body, err := googleapi.WithoutDataWrapper.JSONReader(c.alertfeedback)
	if err != nil {
		return nil, err
	}
	reqHeaders.Set("Content-Type", "application/json")
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v1beta1/alerts/{alertId}/feedback")
	urls += "?" + c.urlParams_.Encode()
	req, err := http.NewRequest("POST", urls, body)
	if err != nil {
		return nil, err
	}
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"alertId": c.alertId,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "alertcenter.alerts.feedback.create" call.
// Exactly one of *AlertFeedback or error will be non-nil. Any non-2xx
// status code is an error. Response headers are in either
// *AlertFeedback.ServerResponse.Header or (if a response was returned
// at all) in error.(*googleapi.Error).Header. Use
// googleapi.IsNotModified to check whether the returned error was
// because http.StatusNotModified was returned.
func (c *AlertsFeedbackCreateCall) Do(opts ...googleapi.CallOption) (*AlertFeedback, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &AlertFeedback{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Creates a new alert feedback.",
	//   "flatPath": "v1beta1/alerts/{alertId}/feedback",
	//   "httpMethod": "POST",
	//   "id": "alertcenter.alerts.feedback.create",
	//   "parameterOrder": [
	//     "alertId"
	//   ],
	//   "parameters": {
	//     "alertId": {
	//       "description": "Required. The identifier of the alert this feedback belongs to.\nReturns a NOT_FOUND error if no such alert.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "customerId": {
	//       "description": "Optional. The unique identifier of the Google account of the customer the\nalert's feedback is associated with. This is obfuscated and not the\nplain customer ID as stored internally. Inferred from the caller identity\nif not provided.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "v1beta1/alerts/{alertId}/feedback",
	//   "request": {
	//     "$ref": "AlertFeedback"
	//   },
	//   "response": {
	//     "$ref": "AlertFeedback"
	//   }
	// }

}

// method id "alertcenter.alerts.feedback.list":

type AlertsFeedbackListCall struct {
	s            *Service
	alertId      string
	urlParams_   gensupport.URLParams
	ifNoneMatch_ string
	ctx_         context.Context
	header_      http.Header
}

// List: Lists all the feedback for an alert.
func (r *AlertsFeedbackService) List(alertId string) *AlertsFeedbackListCall {
	c := &AlertsFeedbackListCall{s: r.s, urlParams_: make(gensupport.URLParams)}
	c.alertId = alertId
	return c
}

// CustomerId sets the optional parameter "customerId": The unique
// identifier of the Google account of the customer the
// alert is associated with. This is obfuscated and not the plain
// customer
// ID as stored internally. Inferred from the caller identity if not
// provided.
func (c *AlertsFeedbackListCall) CustomerId(customerId string) *AlertsFeedbackListCall {
	c.urlParams_.Set("customerId", customerId)
	return c
}

// Fields allows partial responses to be retrieved. See
// https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
// for more information.
func (c *AlertsFeedbackListCall) Fields(s ...googleapi.Field) *AlertsFeedbackListCall {
	c.urlParams_.Set("fields", googleapi.CombineFields(s))
	return c
}

// IfNoneMatch sets the optional parameter which makes the operation
// fail if the object's ETag matches the given value. This is useful for
// getting updates only after the object has changed since the last
// request. Use googleapi.IsNotModified to check whether the response
// error from Do is the result of In-None-Match.
func (c *AlertsFeedbackListCall) IfNoneMatch(entityTag string) *AlertsFeedbackListCall {
	c.ifNoneMatch_ = entityTag
	return c
}

// Context sets the context to be used in this call's Do method. Any
// pending HTTP request will be aborted if the provided context is
// canceled.
func (c *AlertsFeedbackListCall) Context(ctx context.Context) *AlertsFeedbackListCall {
	c.ctx_ = ctx
	return c
}

// Header returns an http.Header that can be modified by the caller to
// add HTTP headers to the request.
func (c *AlertsFeedbackListCall) Header() http.Header {
	if c.header_ == nil {
		c.header_ = make(http.Header)
	}
	return c.header_
}

func (c *AlertsFeedbackListCall) doRequest(alt string) (*http.Response, error) {
	reqHeaders := make(http.Header)
	for k, v := range c.header_ {
		reqHeaders[k] = v
	}
	reqHeaders.Set("User-Agent", c.s.userAgent())
	if c.ifNoneMatch_ != "" {
		reqHeaders.Set("If-None-Match", c.ifNoneMatch_)
	}
	var body io.Reader = nil
	c.urlParams_.Set("alt", alt)
	c.urlParams_.Set("prettyPrint", "false")
	urls := googleapi.ResolveRelative(c.s.BasePath, "v1beta1/alerts/{alertId}/feedback")
	urls += "?" + c.urlParams_.Encode()
	req, err := http.NewRequest("GET", urls, body)
	if err != nil {
		return nil, err
	}
	req.Header = reqHeaders
	googleapi.Expand(req.URL, map[string]string{
		"alertId": c.alertId,
	})
	return gensupport.SendRequest(c.ctx_, c.s.client, req)
}

// Do executes the "alertcenter.alerts.feedback.list" call.
// Exactly one of *ListAlertFeedbackResponse or error will be non-nil.
// Any non-2xx status code is an error. Response headers are in either
// *ListAlertFeedbackResponse.ServerResponse.Header or (if a response
// was returned at all) in error.(*googleapi.Error).Header. Use
// googleapi.IsNotModified to check whether the returned error was
// because http.StatusNotModified was returned.
func (c *AlertsFeedbackListCall) Do(opts ...googleapi.CallOption) (*ListAlertFeedbackResponse, error) {
	gensupport.SetOptions(c.urlParams_, opts...)
	res, err := c.doRequest("json")
	if res != nil && res.StatusCode == http.StatusNotModified {
		if res.Body != nil {
			res.Body.Close()
		}
		return nil, &googleapi.Error{
			Code:   res.StatusCode,
			Header: res.Header,
		}
	}
	if err != nil {
		return nil, err
	}
	defer googleapi.CloseBody(res)
	if err := googleapi.CheckResponse(res); err != nil {
		return nil, err
	}
	ret := &ListAlertFeedbackResponse{
		ServerResponse: googleapi.ServerResponse{
			Header:         res.Header,
			HTTPStatusCode: res.StatusCode,
		},
	}
	target := &ret
	if err := gensupport.DecodeResponse(target, res); err != nil {
		return nil, err
	}
	return ret, nil
	// {
	//   "description": "Lists all the feedback for an alert.",
	//   "flatPath": "v1beta1/alerts/{alertId}/feedback",
	//   "httpMethod": "GET",
	//   "id": "alertcenter.alerts.feedback.list",
	//   "parameterOrder": [
	//     "alertId"
	//   ],
	//   "parameters": {
	//     "alertId": {
	//       "description": "Required. The alert identifier.\nIf the alert does not exist returns a NOT_FOUND error.",
	//       "location": "path",
	//       "required": true,
	//       "type": "string"
	//     },
	//     "customerId": {
	//       "description": "Optional. The unique identifier of the Google account of the customer the\nalert is associated with. This is obfuscated and not the plain customer\nID as stored internally. Inferred from the caller identity if not provided.",
	//       "location": "query",
	//       "type": "string"
	//     }
	//   },
	//   "path": "v1beta1/alerts/{alertId}/feedback",
	//   "response": {
	//     "$ref": "ListAlertFeedbackResponse"
	//   }
	// }

}
