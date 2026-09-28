package service

import (
	"github.com/Wei-Shaw/sub2api/internal/service/basispoints"
)

const ExcelBPSOmitUnsupportedToolsKey = "openai_excel_bps_omit_unsupported_tools"

// IsExcelBPSOmitUnsupportedToolsEnabled is retained for the account editor.
// User requests on a BPS-enabled account always use Basis Points; the key no
// longer chooses between Codex and BPS. The probe still consults
// excelBPSNativeFallbackReason.
func (a *Account) IsExcelBPSOmitUnsupportedToolsEnabled() bool {
	return a.IsExcelBPSEnabled() && a.Extra[ExcelBPSOmitUnsupportedToolsKey] == true
}

// excelBPSNativeFallbackReason reports hosted capabilities the BPS probe
// cannot execute. Ordinary forwarding ignores this and stays on BPS.
func (a *Account) excelBPSNativeFallbackReason(body []byte) string {
	if a.IsExcelBPSOmitUnsupportedToolsEnabled() {
		return ""
	}
	return basispoints.NativeFallbackReason(body)
}
