package database

import "net/netip"

// nullableString конвертирует строку в *string для nullable полей в БД.
// Возвращает nil для пустой строки (маппится в SQL NULL).
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// derefString конвертирует *string обратно в строку.
// Возвращает пустую строку для nil (SQL NULL).
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// parseIPToInet конвертирует строку IP-адреса в *netip.Prefix для записи в INET колонку.
// Возвращает nil для пустой строки (маппится в SQL NULL).
// Использует /32 для IPv4 и /128 для IPv6 для совместимости с тем, как pgx сканирует INET.
func parseIPToInet(s string) (*netip.Prefix, error) {
	if s == "" {
		return nil, nil
	}
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return nil, err
	}

	// Конвертируем в Prefix с полной маской для консистентности с pgx scan
	var prefix netip.Prefix
	if addr.Is4() {
		prefix = netip.PrefixFrom(addr, 32)
	} else {
		prefix = netip.PrefixFrom(addr, 128)
	}
	return &prefix, nil
}

// inetToString конвертирует *netip.Prefix (результат Scan из INET колонки) обратно в строку.
// Возвращает пустую строку для nil (SQL NULL).
func inetToString(p *netip.Prefix) string {
	if p == nil {
		return ""
	}
	return p.Addr().String()
}
