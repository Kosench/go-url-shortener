package cache

import (
	"crypto/md5"
	"fmt"
)

// KeyPrefix - префиксы для разных типов ключей
type KeyPrefix string

const (
	PrefixURL       KeyPrefix = "url"     // url:shortCode
	PrefixShort     KeyPrefix = "short"   // short:hash(originalURL)
	PrefixClicks    KeyPrefix = "clicks"  // clicks:shortCode
	PrefixRateLimit KeyPrefix = "rate"    // rate:clientIP
	PrefixSession   KeyPrefix = "session" // session:sessionID
	PrefixTemp      KeyPrefix = "tmp"     // tmp:uniqueID
)

// KeyBuilder - построитель ключей кэша
type KeyBuilder struct {
	namespace string // Опциональный namespace для multi-tenancy
}

// NewKeyBuilder создает новый построитель ключей
func NewKeyBuilder(namespace string) *KeyBuilder {
	return &KeyBuilder{namespace: namespace}
}

// Build создает ключ с префиксом и опциональным namespace
func (k *KeyBuilder) Build(prefix KeyPrefix, parts ...string) string {
	key := string(prefix)

	if k.namespace != "" {
		key = k.namespace + ":" + key
	}

	for _, part := range parts {
		key += ":" + part
	}

	return key
}

// URL создает ключ для хранения URL по короткому коду
func (k *KeyBuilder) URL(shortCode string) string {
	return k.Build(PrefixURL, shortCode)
}

// ShortCode создает ключ для обратного маппинга (originalURL -> shortCode)
func (k *KeyBuilder) ShortCode(originalURL string) string {
	hash := hashURL(originalURL)
	return k.Build(PrefixShort, hash)
}

// Clicks создает ключ для счетчика кликов
func (k *KeyBuilder) Clicks(shortCode string) string {
	return k.Build(PrefixClicks, shortCode)
}

// RateLimit создает ключ для rate limiting
func (k *KeyBuilder) RateLimit(clientIP string) string {
	return k.Build(PrefixRateLimit, clientIP)
}

// Session создает ключ для сессии
func (k *KeyBuilder) Session(sessionID string) string {
	return k.Build(PrefixSession, sessionID)
}

// Temp создает временный ключ
func (k *KeyBuilder) Temp(id string) string {
	return k.Build(PrefixTemp, id)
}

// Pattern возвращает паттерн для поиска ключей
func (k *KeyBuilder) Pattern(prefix KeyPrefix) string {
	if k.namespace != "" {
		return fmt.Sprintf("%s:%s:*", k.namespace, prefix)
	}
	return fmt.Sprintf("%s:*", prefix)
}

// hashURL создает хэш от URL для использования в ключе
func hashURL(url string) string {
	h := md5.Sum([]byte(url))
	return fmt.Sprintf("%x", h)
}

// DefaultKeyBuilder - построитель ключей по умолчанию
var DefaultKeyBuilder = NewKeyBuilder("")

// Shortcuts для обратной совместимости
var CacheKeys = struct {
	URL       func(string) string
	ShortCode func(string) string
	Clicks    func(string) string
	RateLimit func(string) string
}{
	URL:       DefaultKeyBuilder.URL,
	ShortCode: DefaultKeyBuilder.ShortCode,
	Clicks:    DefaultKeyBuilder.Clicks,
	RateLimit: DefaultKeyBuilder.RateLimit,
}
