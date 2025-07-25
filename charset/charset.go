// Copyright 2015 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package charset

import (
	"slices"
	"strings"

	"github.com/pingcap/errors"
	"github.com/pingcap/log"
	"github.com/abbychau/mysql-parser/mysql"
	"github.com/abbychau/mysql-parser/terror"
	"go.uber.org/zap"
)

var (
	// ErrUnknownCollation is unknown collation.
	ErrUnknownCollation = terror.ClassDDL.NewStd(mysql.ErrUnknownCollation)
	// ErrCollationCharsetMismatch is collation charset mismatch.
	ErrCollationCharsetMismatch = terror.ClassDDL.NewStd(mysql.ErrCollationCharsetMismatch)
)

var (
	// PadSpace is to mark that trailing spaces are insignificant in comparisons
	PadSpace = "PAD SPACE"
	// PadNone is to mark that trailing spaces are significant in comparisons
	PadNone = "NO PAD"
)

// Charset is a charset.
// Now we only support MySQL.
type Charset struct {
	Name             string
	DefaultCollation string
	Collations       map[string]*Collation
	Desc             string
	Maxlen           int
}

// Collation is a collation.
// Now we only support MySQL.
type Collation struct {
	ID           int
	CharsetName  string
	Name         string
	IsDefault    bool
	Sortlen      int
	PadAttribute string
}

var collationsIDMap = make(map[int]*Collation)
var collationsNameMap = make(map[string]*Collation)
var supportedCollations = make([]*Collation, 0, len(supportedCollationNames))

// CharacterSetInfos contains all the supported charsets.
var CharacterSetInfos = map[string]*Charset{
	CharsetUTF8:    {CharsetUTF8, CollationUTF8, make(map[string]*Collation), "UTF-8 Unicode", 3},
	CharsetUTF8MB4: {CharsetUTF8MB4, CollationUTF8MB4, make(map[string]*Collation), "UTF-8 Unicode", 4},
	CharsetASCII:   {CharsetASCII, CollationASCII, make(map[string]*Collation), "US ASCII", 1},
	CharsetLatin1:  {CharsetLatin1, CollationLatin1, make(map[string]*Collation), "Latin1", 1},
	CharsetBin:     {CharsetBin, CollationBin, make(map[string]*Collation), "binary", 1},
	CharsetGBK:     {CharsetGBK, CollationGBKBin, make(map[string]*Collation), "Chinese Internal Code Specification", 2},
	CharsetGB18030: {CharsetGB18030, CollationGB18030Bin, make(map[string]*Collation), "China National Standard GB18030", 4},
}

// All the names supported collations should be in the following table.
var supportedCollationNames = map[string]struct{}{
	CollationUTF8:       {},
	CollationUTF8MB4:    {},
	CollationASCII:      {},
	CollationLatin1:     {},
	CollationBin:        {},
	CollationGBKBin:     {},
	CollationGB18030Bin: {},
}

// TiFlashSupportedCharsets is a map which contains TiFlash supports charsets.
var TiFlashSupportedCharsets = map[string]struct{}{
	CharsetUTF8:    {},
	CharsetUTF8MB4: {},
	CharsetASCII:   {},
	CharsetLatin1:  {},
	CharsetBin:     {},
}

// GetSupportedCharsets gets descriptions for all charsets supported so far.
func GetSupportedCharsets() []*Charset {
	charsets := make([]*Charset, 0, len(CharacterSetInfos))
	for _, ch := range CharacterSetInfos {
		charsets = append(charsets, ch)
	}

	// sort charset by name.
	slices.SortFunc(charsets, func(i, j *Charset) int {
		return strings.Compare(i.Name, j.Name)
	})
	return charsets
}

// GetSupportedCollations gets information for all collations supported so far.
func GetSupportedCollations() []*Collation {
	return supportedCollations
}

// ValidCharsetAndCollation checks the charset and the collation validity
// and returns a boolean.
func ValidCharsetAndCollation(cs string, co string) bool {
	// We will use utf8 as a default charset.
	if cs == "" || cs == CharsetUTF8MB3 {
		cs = CharsetUTF8
	}
	chs, err := GetCharsetInfo(cs)
	if err != nil {
		return false
	}

	if co == "" {
		return true
	}
	co = utf8Alias(strings.ToLower(co))
	_, ok := chs.Collations[co]
	return ok
}

// GetDefaultCollationLegacy is compatible with the charset support in old version parser.
func GetDefaultCollationLegacy(charset string) (string, error) {
	switch strings.ToLower(charset) {
	case CharsetUTF8MB3:
		return GetDefaultCollation(CharsetUTF8)
	case CharsetUTF8, CharsetUTF8MB4, CharsetASCII, CharsetLatin1, CharsetBin:
		return GetDefaultCollation(charset)
	default:
		return "", errors.Errorf("Unknown charset %s", charset)
	}
}

// GetDefaultCollation returns the default collation for charset.
func GetDefaultCollation(charset string) (string, error) {
	cs, err := GetCharsetInfo(charset)
	if err != nil {
		return "", err
	}
	return cs.DefaultCollation, nil
}

// GetDefaultCharsetAndCollate returns the default charset and collation.
func GetDefaultCharsetAndCollate() (defaultCharset string, defaultCollationName string) {
	return mysql.DefaultCharset, mysql.DefaultCollationName
}

// GetCharsetInfo returns charset and collation for cs as name.
func GetCharsetInfo(cs string) (*Charset, error) {
	if strings.ToLower(cs) == CharsetUTF8MB3 {
		cs = CharsetUTF8
	}

	if c, ok := CharacterSetInfos[strings.ToLower(cs)]; ok {
		return c, nil
	}

	if c, ok := charsets[strings.ToLower(cs)]; ok {
		return c, errors.Errorf("Unsupported charset %s", cs)
	}

	return nil, errors.Errorf("Unknown charset %s", cs)
}

// GetCharsetInfoByID returns charset and collation for id as cs_number.
func GetCharsetInfoByID(coID int) (charsetStr string, collateStr string, err error) {
	if coID == mysql.DefaultCollationID {
		return mysql.DefaultCharset, mysql.DefaultCollationName, nil
	}
	if collation, ok := collationsIDMap[coID]; ok {
		return collation.CharsetName, collation.Name, nil
	}

	log.Warn(
		"unable to get collation name from collation ID, return default charset and collation instead",
		zap.Int("ID", coID),
		zap.Stack("stack"))
	return mysql.DefaultCharset, mysql.DefaultCollationName, errors.Errorf("Unknown collation id %d", coID)
}

func utf8Alias(csname string) string {
	switch csname {
	case "utf8mb3_bin":
		csname = "utf8_bin"
	case "utf8mb3_unicode_ci":
		csname = "utf8_unicode_ci"
	case "utf8mb3_general_ci":
		csname = "utf8_general_ci"
	default:
	}
	return csname
}

// GetCollationByName returns the collation by name.
func GetCollationByName(name string) (*Collation, error) {
	csname := utf8Alias(strings.ToLower(name))
	collation, ok := collationsNameMap[csname]
	if !ok {
		return nil, ErrUnknownCollation.GenWithStackByArgs(name)
	}
	return collation, nil
}

// GetCollationByID returns collations by given id.
func GetCollationByID(id int) (*Collation, error) {
	collation, ok := collationsIDMap[id]
	if !ok {
		return nil, errors.Errorf("Unknown collation id %d", id)
	}

	return collation, nil
}

const (
	// CollationBin is the default collation for CharsetBin.
	CollationBin = "binary"
	// CollationUTF8 is the default collation for CharsetUTF8.
	CollationUTF8 = "utf8_bin"
	// CollationUTF8MB4 is the default collation for CharsetUTF8MB4.
	CollationUTF8MB4 = "utf8mb4_bin"
	// CollationASCII is the default collation for CharsetACSII.
	CollationASCII = "ascii_bin"
	// CollationLatin1 is the default collation for CharsetLatin1.
	CollationLatin1 = "latin1_bin"
	// CollationGBKBin is the default collation for CharsetGBK when new collation is disabled.
	CollationGBKBin = "gbk_bin"
	// CollationGBKChineseCI is the default collation for CharsetGBK when new collation is enabled.
	CollationGBKChineseCI = "gbk_chinese_ci"
	// CollationGB18030Bin is the default collation for CharsetGB18030 when new collation is disabled.
	CollationGB18030Bin = "gb18030_bin"
	// CollationGB18030ChineseCI is the default collation for CharsetGB18030 when new collation is enabled.
	CollationGB18030ChineseCI = "gb18030_chinese_ci"
)

const (
	// CharsetASCII is a subset of UTF8.
	CharsetASCII = "ascii"
	// CharsetBin is used for marking binary charset.
	CharsetBin = "binary"
	// CharsetLatin1 is a single byte charset.
	CharsetLatin1 = "latin1"
	// CharsetUTF8 is the default charset for string types.
	CharsetUTF8 = "utf8"
	// CharsetUTF8MB3 is 3 bytes utf8, a MySQL legacy encoding. "utf8" and "utf8mb3" are aliases.
	CharsetUTF8MB3 = "utf8mb3"
	// CharsetUTF8MB4 represents 4 bytes utf8, which works the same way as utf8 in Go.
	CharsetUTF8MB4 = "utf8mb4"
	// CharsetGB18030 represents 4 bytes gb18030.
	CharsetGB18030 = "gb18030"
	//revive:disable:exported
	CharsetARMSCII8 = "armscii8"
	CharsetBig5     = "big5"
	CharsetCP1250   = "cp1250"
	CharsetCP1251   = "cp1251"
	CharsetCP1256   = "cp1256"
	CharsetCP1257   = "cp1257"
	CharsetCP850    = "cp850"
	CharsetCP852    = "cp852"
	CharsetCP866    = "cp866"
	CharsetCP932    = "cp932"
	CharsetDEC8     = "dec8"
	CharsetEUCJPMS  = "eucjpms"
	CharsetEUCKR    = "euckr"
	CharsetGB2312   = "gb2312"
	CharsetGBK      = "gbk"
	CharsetGEOSTD8  = "geostd8"
	CharsetGreek    = "greek"
	CharsetHebrew   = "hebrew"
	CharsetHP8      = "hp8"
	CharsetKEYBCS2  = "keybcs2"
	CharsetKOI8R    = "koi8r"
	CharsetKOI8U    = "koi8u"
	CharsetLatin2   = "latin2"
	CharsetLatin5   = "latin5"
	CharsetLatin7   = "latin7"
	CharsetMacCE    = "macce"
	CharsetMacRoman = "macroman"
	CharsetSJIS     = "sjis"
	CharsetSWE7     = "swe7"
	CharsetTIS620   = "tis620"
	CharsetUCS2     = "ucs2"
	CharsetUJIS     = "ujis"
	CharsetUTF16    = "utf16"
	CharsetUTF16LE  = "utf16le"
	CharsetUTF32    = "utf32"
	//revive:enable:exported
)

var charsets = map[string]*Charset{
	CharsetARMSCII8: {Name: CharsetARMSCII8, Maxlen: 1, DefaultCollation: "armscii8_general_ci", Desc: "ARMSCII-8 Armenian", Collations: make(map[string]*Collation)},
	CharsetASCII:    {Name: CharsetASCII, Maxlen: 1, DefaultCollation: "ascii_general_ci", Desc: "US ASCII", Collations: make(map[string]*Collation)},
	CharsetBig5:     {Name: CharsetBig5, Maxlen: 2, DefaultCollation: "big5_chinese_ci", Desc: "Big5 Traditional Chinese", Collations: make(map[string]*Collation)},
	CharsetBin:      {Name: CharsetBin, Maxlen: 1, DefaultCollation: "binary", Desc: "Binary pseudo charset", Collations: make(map[string]*Collation)},
	CharsetCP1250:   {Name: CharsetCP1250, Maxlen: 1, DefaultCollation: "cp1250_general_ci", Desc: "Windows Central European", Collations: make(map[string]*Collation)},
	CharsetCP1251:   {Name: CharsetCP1251, Maxlen: 1, DefaultCollation: "cp1251_general_ci", Desc: "Windows Cyrillic", Collations: make(map[string]*Collation)},
	CharsetCP1256:   {Name: CharsetCP1256, Maxlen: 1, DefaultCollation: "cp1256_general_ci", Desc: "Windows Arabic", Collations: make(map[string]*Collation)},
	CharsetCP1257:   {Name: CharsetCP1257, Maxlen: 1, DefaultCollation: "cp1257_general_ci", Desc: "Windows Baltic", Collations: make(map[string]*Collation)},
	CharsetCP850:    {Name: CharsetCP850, Maxlen: 1, DefaultCollation: "cp850_general_ci", Desc: "DOS West European", Collations: make(map[string]*Collation)},
	CharsetCP852:    {Name: CharsetCP852, Maxlen: 1, DefaultCollation: "cp852_general_ci", Desc: "DOS Central European", Collations: make(map[string]*Collation)},
	CharsetCP866:    {Name: CharsetCP866, Maxlen: 1, DefaultCollation: "cp866_general_ci", Desc: "DOS Russian", Collations: make(map[string]*Collation)},
	CharsetCP932:    {Name: CharsetCP932, Maxlen: 2, DefaultCollation: "cp932_japanese_ci", Desc: "SJIS for Windows Japanese", Collations: make(map[string]*Collation)},
	CharsetDEC8:     {Name: CharsetDEC8, Maxlen: 1, DefaultCollation: "dec8_swedish_ci", Desc: "DEC West European", Collations: make(map[string]*Collation)},
	CharsetEUCJPMS:  {Name: CharsetEUCJPMS, Maxlen: 3, DefaultCollation: "eucjpms_japanese_ci", Desc: "UJIS for Windows Japanese", Collations: make(map[string]*Collation)},
	CharsetEUCKR:    {Name: CharsetEUCKR, Maxlen: 2, DefaultCollation: "euckr_korean_ci", Desc: "EUC-KR Korean", Collations: make(map[string]*Collation)},
	CharsetGB18030:  {Name: CharsetGB18030, Maxlen: 4, DefaultCollation: "gb18030_chinese_ci", Desc: "China National Standard GB18030", Collations: make(map[string]*Collation)},
	CharsetGB2312:   {Name: CharsetGB2312, Maxlen: 2, DefaultCollation: "gb2312_chinese_ci", Desc: "GB2312 Simplified Chinese", Collations: make(map[string]*Collation)},
	CharsetGBK:      {Name: CharsetGBK, Maxlen: 2, DefaultCollation: "gbk_chinese_ci", Desc: "GBK Simplified Chinese", Collations: make(map[string]*Collation)},
	CharsetGEOSTD8:  {Name: CharsetGEOSTD8, Maxlen: 1, DefaultCollation: "geostd8_general_ci", Desc: "GEOSTD8 Georgian", Collations: make(map[string]*Collation)},
	CharsetGreek:    {Name: CharsetGreek, Maxlen: 1, DefaultCollation: "greek_general_ci", Desc: "ISO 8859-7 Greek", Collations: make(map[string]*Collation)},
	CharsetHebrew:   {Name: CharsetHebrew, Maxlen: 1, DefaultCollation: "hebrew_general_ci", Desc: "ISO 8859-8 Hebrew", Collations: make(map[string]*Collation)},
	CharsetHP8:      {Name: CharsetHP8, Maxlen: 1, DefaultCollation: "hp8_english_ci", Desc: "HP West European", Collations: make(map[string]*Collation)},
	CharsetKEYBCS2:  {Name: CharsetKEYBCS2, Maxlen: 1, DefaultCollation: "keybcs2_general_ci", Desc: "DOS Kamenicky Czech-Slovak", Collations: make(map[string]*Collation)},
	CharsetKOI8R:    {Name: CharsetKOI8R, Maxlen: 1, DefaultCollation: "koi8u_general_ci", Desc: "KOI8-U Ukrainian", Collations: make(map[string]*Collation)},
	CharsetKOI8U:    {Name: CharsetKOI8U, Maxlen: 1, DefaultCollation: "koi8r_general_ci", Desc: "KOI8-R Relcom Russian", Collations: make(map[string]*Collation)},
	CharsetLatin1:   {Name: CharsetLatin1, Maxlen: 1, DefaultCollation: "latin1_swedish_ci", Desc: "cp1252 West European", Collations: make(map[string]*Collation)},
	CharsetLatin2:   {Name: CharsetLatin2, Maxlen: 1, DefaultCollation: "latin2_general_ci", Desc: "ISO 8859-2 Central European", Collations: make(map[string]*Collation)},
	CharsetLatin5:   {Name: CharsetLatin5, Maxlen: 1, DefaultCollation: "latin5_turkish_ci", Desc: "ISO 8859-9 Turkish", Collations: make(map[string]*Collation)},
	CharsetLatin7:   {Name: CharsetLatin7, Maxlen: 1, DefaultCollation: "latin7_general_ci", Desc: "ISO 8859-13 Baltic", Collations: make(map[string]*Collation)},
	CharsetMacCE:    {Name: CharsetMacCE, Maxlen: 1, DefaultCollation: "macce_general_ci", Desc: "Mac Central European", Collations: make(map[string]*Collation)},
	CharsetMacRoman: {Name: CharsetMacRoman, Maxlen: 1, DefaultCollation: "macroman_general_ci", Desc: "Mac West European", Collations: make(map[string]*Collation)},
	CharsetSJIS:     {Name: CharsetSJIS, Maxlen: 2, DefaultCollation: "sjis_japanese_ci", Desc: "Shift-JIS Japanese", Collations: make(map[string]*Collation)},
	CharsetSWE7:     {Name: CharsetSWE7, Maxlen: 1, DefaultCollation: "swe7_swedish_ci", Desc: "7bit Swedish", Collations: make(map[string]*Collation)},
	CharsetTIS620:   {Name: CharsetTIS620, Maxlen: 1, DefaultCollation: "tis620_thai_ci", Desc: "TIS620 Thai", Collations: make(map[string]*Collation)},
	CharsetUCS2:     {Name: CharsetUCS2, Maxlen: 2, DefaultCollation: "ucs2_general_ci", Desc: "UCS-2 Unicode", Collations: make(map[string]*Collation)},
	CharsetUJIS:     {Name: CharsetUJIS, Maxlen: 3, DefaultCollation: "ujis_japanese_ci", Desc: "EUC-JP Japanese", Collations: make(map[string]*Collation)},
	CharsetUTF16:    {Name: CharsetUTF16, Maxlen: 4, DefaultCollation: "utf16_general_ci", Desc: "UTF-16 Unicode", Collations: make(map[string]*Collation)},
	CharsetUTF16LE:  {Name: CharsetUTF16LE, Maxlen: 4, DefaultCollation: "utf16le_general_ci", Desc: "UTF-16LE Unicode", Collations: make(map[string]*Collation)},
	CharsetUTF32:    {Name: CharsetUTF32, Maxlen: 4, DefaultCollation: "utf32_general_ci", Desc: "UTF-32 Unicode", Collations: make(map[string]*Collation)},
	CharsetUTF8:     {Name: CharsetUTF8, Maxlen: 3, DefaultCollation: "utf8_general_ci", Desc: "UTF-8 Unicode", Collations: make(map[string]*Collation)},
	CharsetUTF8MB4:  {Name: CharsetUTF8MB4, Maxlen: 4, DefaultCollation: "utf8mb4_0900_ai_ci", Desc: "UTF-8 Unicode", Collations: make(map[string]*Collation)},
}

var collations = []*Collation{
	{1, "big5", "big5_chinese_ci", true, 1, PadSpace},
	{2, "latin2", "latin2_czech_cs", false, 1, PadSpace},
	{3, "dec8", "dec8_swedish_ci", true, 1, PadSpace},
	{4, "cp850", "cp850_general_ci", true, 1, PadSpace},
	{5, "latin1", "latin1_german1_ci", false, 1, PadSpace},
	{6, "hp8", "hp8_english_ci", true, 1, PadSpace},
	{7, "koi8r", "koi8r_general_ci", true, 1, PadSpace},
	{8, "latin1", "latin1_swedish_ci", false, 1, PadSpace},
	{9, "latin2", "latin2_general_ci", true, 1, PadSpace},
	{10, "swe7", "swe7_swedish_ci", true, 1, PadSpace},
	{11, "ascii", "ascii_general_ci", false, 1, PadSpace},
	{12, "ujis", "ujis_japanese_ci", true, 1, PadSpace},
	{13, "sjis", "sjis_japanese_ci", true, 1, PadSpace},
	{14, "cp1251", "cp1251_bulgarian_ci", false, 1, PadSpace},
	{15, "latin1", "latin1_danish_ci", false, 1, PadSpace},
	{16, "hebrew", "hebrew_general_ci", true, 1, PadSpace},
	{18, "tis620", "tis620_thai_ci", true, 1, PadSpace},
	{19, "euckr", "euckr_korean_ci", true, 1, PadSpace},
	{20, "latin7", "latin7_estonian_cs", false, 1, PadSpace},
	{21, "latin2", "latin2_hungarian_ci", false, 1, PadSpace},
	{22, "koi8u", "koi8u_general_ci", true, 1, PadSpace},
	{23, "cp1251", "cp1251_ukrainian_ci", false, 1, PadSpace},
	{24, "gb2312", "gb2312_chinese_ci", true, 1, PadSpace},
	{25, "greek", "greek_general_ci", true, 1, PadSpace},
	{26, "cp1250", "cp1250_general_ci", true, 1, PadSpace},
	{27, "latin2", "latin2_croatian_ci", false, 1, PadSpace},
	{28, "gbk", "gbk_chinese_ci", false, 1, PadSpace},
	{29, "cp1257", "cp1257_lithuanian_ci", false, 1, PadSpace},
	{30, "latin5", "latin5_turkish_ci", true, 1, PadSpace},
	{31, "latin1", "latin1_german2_ci", false, 1, PadSpace},
	{32, "armscii8", "armscii8_general_ci", true, 1, PadSpace},
	{33, "utf8", "utf8_general_ci", false, 1, PadSpace},
	{34, "cp1250", "cp1250_czech_cs", false, 1, PadSpace},
	{35, "ucs2", "ucs2_general_ci", true, 1, PadSpace},
	{36, "cp866", "cp866_general_ci", true, 1, PadSpace},
	{37, "keybcs2", "keybcs2_general_ci", true, 1, PadSpace},
	{38, "macce", "macce_general_ci", true, 1, PadSpace},
	{39, "macroman", "macroman_general_ci", true, 1, PadSpace},
	{40, "cp852", "cp852_general_ci", true, 1, PadSpace},
	{41, "latin7", "latin7_general_ci", true, 1, PadSpace},
	{42, "latin7", "latin7_general_cs", false, 1, PadSpace},
	{43, "macce", "macce_bin", false, 1, PadSpace},
	{44, "cp1250", "cp1250_croatian_ci", false, 1, PadSpace},
	{45, "utf8mb4", "utf8mb4_general_ci", false, 1, PadSpace},
	{46, "utf8mb4", "utf8mb4_bin", true, 1, PadSpace},
	{47, "latin1", "latin1_bin", true, 1, PadSpace},
	{48, "latin1", "latin1_general_ci", false, 1, PadSpace},
	{49, "latin1", "latin1_general_cs", false, 1, PadSpace},
	{50, "cp1251", "cp1251_bin", false, 1, PadSpace},
	{51, "cp1251", "cp1251_general_ci", true, 1, PadSpace},
	{52, "cp1251", "cp1251_general_cs", false, 1, PadSpace},
	{53, "macroman", "macroman_bin", false, 1, PadSpace},
	{54, "utf16", "utf16_general_ci", true, 1, PadSpace},
	{55, "utf16", "utf16_bin", false, 1, PadSpace},
	{56, "utf16le", "utf16le_general_ci", true, 1, PadSpace},
	{57, "cp1256", "cp1256_general_ci", true, 1, PadSpace},
	{58, "cp1257", "cp1257_bin", false, 1, PadSpace},
	{59, "cp1257", "cp1257_general_ci", true, 1, PadSpace},
	{60, "utf32", "utf32_general_ci", true, 1, PadSpace},
	{61, "utf32", "utf32_bin", false, 1, PadSpace},
	{62, "utf16le", "utf16le_bin", false, 1, PadSpace},
	{63, "binary", "binary", true, 1, PadNone},
	{64, "armscii8", "armscii8_bin", false, 1, PadSpace},
	{65, "ascii", "ascii_bin", true, 1, PadSpace},
	{66, "cp1250", "cp1250_bin", false, 1, PadSpace},
	{67, "cp1256", "cp1256_bin", false, 1, PadSpace},
	{68, "cp866", "cp866_bin", false, 1, PadSpace},
	{69, "dec8", "dec8_bin", false, 1, PadSpace},
	{70, "greek", "greek_bin", false, 1, PadSpace},
	{71, "hebrew", "hebrew_bin", false, 1, PadSpace},
	{72, "hp8", "hp8_bin", false, 1, PadSpace},
	{73, "keybcs2", "keybcs2_bin", false, 1, PadSpace},
	{74, "koi8r", "koi8r_bin", false, 1, PadSpace},
	{75, "koi8u", "koi8u_bin", false, 1, PadSpace},
	{76, "utf8", "utf8_tolower_ci", false, 1, PadNone},
	{77, "latin2", "latin2_bin", false, 1, PadSpace},
	{78, "latin5", "latin5_bin", false, 1, PadSpace},
	{79, "latin7", "latin7_bin", false, 1, PadSpace},
	{80, "cp850", "cp850_bin", false, 1, PadSpace},
	{81, "cp852", "cp852_bin", false, 1, PadSpace},
	{82, "swe7", "swe7_bin", false, 1, PadSpace},
	{83, "utf8", "utf8_bin", true, 1, PadSpace},
	{84, "big5", "big5_bin", false, 1, PadSpace},
	{85, "euckr", "euckr_bin", false, 1, PadSpace},
	{86, "gb2312", "gb2312_bin", false, 1, PadSpace},
	{87, "gbk", "gbk_bin", true, 1, PadSpace},
	{88, "sjis", "sjis_bin", false, 1, PadSpace},
	{89, "tis620", "tis620_bin", false, 1, PadSpace},
	{90, "ucs2", "ucs2_bin", false, 1, PadSpace},
	{91, "ujis", "ujis_bin", false, 1, PadSpace},
	{92, "geostd8", "geostd8_general_ci", true, 1, PadSpace},
	{93, "geostd8", "geostd8_bin", false, 1, PadSpace},
	{94, "latin1", "latin1_spanish_ci", false, 1, PadSpace},
	{95, "cp932", "cp932_japanese_ci", true, 1, PadSpace},
	{96, "cp932", "cp932_bin", false, 1, PadSpace},
	{97, "eucjpms", "eucjpms_japanese_ci", true, 1, PadSpace},
	{98, "eucjpms", "eucjpms_bin", false, 1, PadSpace},
	{99, "cp1250", "cp1250_polish_ci", false, 1, PadSpace},
	{101, "utf16", "utf16_unicode_ci", false, 1, PadSpace},
	{102, "utf16", "utf16_icelandic_ci", false, 1, PadSpace},
	{103, "utf16", "utf16_latvian_ci", false, 1, PadSpace},
	{104, "utf16", "utf16_romanian_ci", false, 1, PadSpace},
	{105, "utf16", "utf16_slovenian_ci", false, 1, PadSpace},
	{106, "utf16", "utf16_polish_ci", false, 1, PadSpace},
	{107, "utf16", "utf16_estonian_ci", false, 1, PadSpace},
	{108, "utf16", "utf16_spanish_ci", false, 1, PadSpace},
	{109, "utf16", "utf16_swedish_ci", false, 1, PadSpace},
	{110, "utf16", "utf16_turkish_ci", false, 1, PadSpace},
	{111, "utf16", "utf16_czech_ci", false, 1, PadSpace},
	{112, "utf16", "utf16_danish_ci", false, 1, PadSpace},
	{113, "utf16", "utf16_lithuanian_ci", false, 1, PadSpace},
	{114, "utf16", "utf16_slovak_ci", false, 1, PadSpace},
	{115, "utf16", "utf16_spanish2_ci", false, 1, PadSpace},
	{116, "utf16", "utf16_roman_ci", false, 1, PadSpace},
	{117, "utf16", "utf16_persian_ci", false, 1, PadSpace},
	{118, "utf16", "utf16_esperanto_ci", false, 1, PadSpace},
	{119, "utf16", "utf16_hungarian_ci", false, 1, PadSpace},
	{120, "utf16", "utf16_sinhala_ci", false, 1, PadSpace},
	{121, "utf16", "utf16_german2_ci", false, 1, PadSpace},
	{122, "utf16", "utf16_croatian_ci", false, 1, PadSpace},
	{123, "utf16", "utf16_unicode_520_ci", false, 1, PadSpace},
	{124, "utf16", "utf16_vietnamese_ci", false, 1, PadSpace},
	{128, "ucs2", "ucs2_unicode_ci", false, 1, PadSpace},
	{129, "ucs2", "ucs2_icelandic_ci", false, 1, PadSpace},
	{130, "ucs2", "ucs2_latvian_ci", false, 1, PadSpace},
	{131, "ucs2", "ucs2_romanian_ci", false, 1, PadSpace},
	{132, "ucs2", "ucs2_slovenian_ci", false, 1, PadSpace},
	{133, "ucs2", "ucs2_polish_ci", false, 1, PadSpace},
	{134, "ucs2", "ucs2_estonian_ci", false, 1, PadSpace},
	{135, "ucs2", "ucs2_spanish_ci", false, 1, PadSpace},
	{136, "ucs2", "ucs2_swedish_ci", false, 1, PadSpace},
	{137, "ucs2", "ucs2_turkish_ci", false, 1, PadSpace},
	{138, "ucs2", "ucs2_czech_ci", false, 1, PadSpace},
	{139, "ucs2", "ucs2_danish_ci", false, 1, PadSpace},
	{140, "ucs2", "ucs2_lithuanian_ci", false, 1, PadSpace},
	{141, "ucs2", "ucs2_slovak_ci", false, 1, PadSpace},
	{142, "ucs2", "ucs2_spanish2_ci", false, 1, PadSpace},
	{143, "ucs2", "ucs2_roman_ci", false, 1, PadSpace},
	{144, "ucs2", "ucs2_persian_ci", false, 1, PadSpace},
	{145, "ucs2", "ucs2_esperanto_ci", false, 1, PadSpace},
	{146, "ucs2", "ucs2_hungarian_ci", false, 1, PadSpace},
	{147, "ucs2", "ucs2_sinhala_ci", false, 1, PadSpace},
	{148, "ucs2", "ucs2_german2_ci", false, 1, PadSpace},
	{149, "ucs2", "ucs2_croatian_ci", false, 1, PadSpace},
	{150, "ucs2", "ucs2_unicode_520_ci", false, 1, PadSpace},
	{151, "ucs2", "ucs2_vietnamese_ci", false, 1, PadSpace},
	{159, "ucs2", "ucs2_general_mysql500_ci", false, 1, PadSpace},
	{160, "utf32", "utf32_unicode_ci", false, 1, PadSpace},
	{161, "utf32", "utf32_icelandic_ci", false, 1, PadSpace},
	{162, "utf32", "utf32_latvian_ci", false, 1, PadSpace},
	{163, "utf32", "utf32_romanian_ci", false, 1, PadSpace},
	{164, "utf32", "utf32_slovenian_ci", false, 1, PadSpace},
	{165, "utf32", "utf32_polish_ci", false, 1, PadSpace},
	{166, "utf32", "utf32_estonian_ci", false, 1, PadSpace},
	{167, "utf32", "utf32_spanish_ci", false, 1, PadSpace},
	{168, "utf32", "utf32_swedish_ci", false, 1, PadSpace},
	{169, "utf32", "utf32_turkish_ci", false, 1, PadSpace},
	{170, "utf32", "utf32_czech_ci", false, 1, PadSpace},
	{171, "utf32", "utf32_danish_ci", false, 1, PadSpace},
	{172, "utf32", "utf32_lithuanian_ci", false, 1, PadSpace},
	{173, "utf32", "utf32_slovak_ci", false, 1, PadSpace},
	{174, "utf32", "utf32_spanish2_ci", false, 1, PadSpace},
	{175, "utf32", "utf32_roman_ci", false, 1, PadSpace},
	{176, "utf32", "utf32_persian_ci", false, 1, PadSpace},
	{177, "utf32", "utf32_esperanto_ci", false, 1, PadSpace},
	{178, "utf32", "utf32_hungarian_ci", false, 1, PadSpace},
	{179, "utf32", "utf32_sinhala_ci", false, 1, PadSpace},
	{180, "utf32", "utf32_german2_ci", false, 1, PadSpace},
	{181, "utf32", "utf32_croatian_ci", false, 1, PadSpace},
	{182, "utf32", "utf32_unicode_520_ci", false, 1, PadSpace},
	{183, "utf32", "utf32_vietnamese_ci", false, 1, PadSpace},
	{192, "utf8", "utf8_unicode_ci", false, 8, PadSpace},
	{193, "utf8", "utf8_icelandic_ci", false, 1, PadNone},
	{194, "utf8", "utf8_latvian_ci", false, 1, PadNone},
	{195, "utf8", "utf8_romanian_ci", false, 1, PadNone},
	{196, "utf8", "utf8_slovenian_ci", false, 1, PadNone},
	{197, "utf8", "utf8_polish_ci", false, 1, PadNone},
	{198, "utf8", "utf8_estonian_ci", false, 1, PadNone},
	{199, "utf8", "utf8_spanish_ci", false, 1, PadNone},
	{200, "utf8", "utf8_swedish_ci", false, 1, PadNone},
	{201, "utf8", "utf8_turkish_ci", false, 1, PadNone},
	{202, "utf8", "utf8_czech_ci", false, 1, PadNone},
	{203, "utf8", "utf8_danish_ci", false, 1, PadNone},
	{204, "utf8", "utf8_lithuanian_ci", false, 1, PadNone},
	{205, "utf8", "utf8_slovak_ci", false, 1, PadNone},
	{206, "utf8", "utf8_spanish2_ci", false, 1, PadNone},
	{207, "utf8", "utf8_roman_ci", false, 1, PadNone},
	{208, "utf8", "utf8_persian_ci", false, 1, PadNone},
	{209, "utf8", "utf8_esperanto_ci", false, 1, PadNone},
	{210, "utf8", "utf8_hungarian_ci", false, 1, PadNone},
	{211, "utf8", "utf8_sinhala_ci", false, 1, PadNone},
	{212, "utf8", "utf8_german2_ci", false, 1, PadNone},
	{213, "utf8", "utf8_croatian_ci", false, 1, PadNone},
	{214, "utf8", "utf8_unicode_520_ci", false, 1, PadNone},
	{215, "utf8", "utf8_vietnamese_ci", false, 1, PadNone},
	{223, "utf8", "utf8_general_mysql500_ci", false, 1, PadNone},
	{224, "utf8mb4", "utf8mb4_unicode_ci", false, 8, PadSpace},
	{225, "utf8mb4", "utf8mb4_icelandic_ci", false, 1, PadSpace},
	{226, "utf8mb4", "utf8mb4_latvian_ci", false, 1, PadSpace},
	{227, "utf8mb4", "utf8mb4_romanian_ci", false, 1, PadSpace},
	{228, "utf8mb4", "utf8mb4_slovenian_ci", false, 1, PadSpace},
	{229, "utf8mb4", "utf8mb4_polish_ci", false, 1, PadSpace},
	{230, "utf8mb4", "utf8mb4_estonian_ci", false, 1, PadSpace},
	{231, "utf8mb4", "utf8mb4_spanish_ci", false, 1, PadSpace},
	{232, "utf8mb4", "utf8mb4_swedish_ci", false, 1, PadSpace},
	{233, "utf8mb4", "utf8mb4_turkish_ci", false, 1, PadSpace},
	{234, "utf8mb4", "utf8mb4_czech_ci", false, 1, PadSpace},
	{235, "utf8mb4", "utf8mb4_danish_ci", false, 1, PadSpace},
	{236, "utf8mb4", "utf8mb4_lithuanian_ci", false, 1, PadSpace},
	{237, "utf8mb4", "utf8mb4_slovak_ci", false, 1, PadSpace},
	{238, "utf8mb4", "utf8mb4_spanish2_ci", false, 1, PadSpace},
	{239, "utf8mb4", "utf8mb4_roman_ci", false, 1, PadSpace},
	{240, "utf8mb4", "utf8mb4_persian_ci", false, 1, PadSpace},
	{241, "utf8mb4", "utf8mb4_esperanto_ci", false, 1, PadSpace},
	{242, "utf8mb4", "utf8mb4_hungarian_ci", false, 1, PadSpace},
	{243, "utf8mb4", "utf8mb4_sinhala_ci", false, 1, PadSpace},
	{244, "utf8mb4", "utf8mb4_german2_ci", false, 1, PadSpace},
	{245, "utf8mb4", "utf8mb4_croatian_ci", false, 1, PadSpace},
	{246, "utf8mb4", "utf8mb4_unicode_520_ci", false, 1, PadSpace},
	{247, "utf8mb4", "utf8mb4_vietnamese_ci", false, 1, PadSpace},
	{248, "gb18030", "gb18030_chinese_ci", false, 1, PadSpace},
	{249, "gb18030", "gb18030_bin", true, 1, PadSpace},
	{250, "gb18030", "gb18030_unicode_520_ci", false, 1, PadSpace},
	{255, "utf8mb4", "utf8mb4_0900_ai_ci", false, 0, PadNone},
	{256, "utf8mb4", "utf8mb4_de_pb_0900_ai_ci", false, 1, PadNone},
	{257, "utf8mb4", "utf8mb4_is_0900_ai_ci", false, 1, PadNone},
	{258, "utf8mb4", "utf8mb4_lv_0900_ai_ci", false, 1, PadNone},
	{259, "utf8mb4", "utf8mb4_ro_0900_ai_ci", false, 1, PadNone},
	{260, "utf8mb4", "utf8mb4_sl_0900_ai_ci", false, 1, PadNone},
	{261, "utf8mb4", "utf8mb4_pl_0900_ai_ci", false, 1, PadNone},
	{262, "utf8mb4", "utf8mb4_et_0900_ai_ci", false, 1, PadNone},
	{263, "utf8mb4", "utf8mb4_es_0900_ai_ci", false, 1, PadNone},
	{264, "utf8mb4", "utf8mb4_sv_0900_ai_ci", false, 1, PadNone},
	{265, "utf8mb4", "utf8mb4_tr_0900_ai_ci", false, 1, PadNone},
	{266, "utf8mb4", "utf8mb4_cs_0900_ai_ci", false, 1, PadNone},
	{267, "utf8mb4", "utf8mb4_da_0900_ai_ci", false, 1, PadNone},
	{268, "utf8mb4", "utf8mb4_lt_0900_ai_ci", false, 1, PadNone},
	{269, "utf8mb4", "utf8mb4_sk_0900_ai_ci", false, 1, PadNone},
	{270, "utf8mb4", "utf8mb4_es_trad_0900_ai_ci", false, 1, PadNone},
	{271, "utf8mb4", "utf8mb4_la_0900_ai_ci", false, 1, PadNone},
	{273, "utf8mb4", "utf8mb4_eo_0900_ai_ci", false, 1, PadNone},
	{274, "utf8mb4", "utf8mb4_hu_0900_ai_ci", false, 1, PadNone},
	{275, "utf8mb4", "utf8mb4_hr_0900_ai_ci", false, 1, PadNone},
	{277, "utf8mb4", "utf8mb4_vi_0900_ai_ci", false, 1, PadNone},
	{278, "utf8mb4", "utf8mb4_0900_as_cs", false, 1, PadNone},
	{279, "utf8mb4", "utf8mb4_de_pb_0900_as_cs", false, 1, PadNone},
	{280, "utf8mb4", "utf8mb4_is_0900_as_cs", false, 1, PadNone},
	{281, "utf8mb4", "utf8mb4_lv_0900_as_cs", false, 1, PadNone},
	{282, "utf8mb4", "utf8mb4_ro_0900_as_cs", false, 1, PadNone},
	{283, "utf8mb4", "utf8mb4_sl_0900_as_cs", false, 1, PadNone},
	{284, "utf8mb4", "utf8mb4_pl_0900_as_cs", false, 1, PadNone},
	{285, "utf8mb4", "utf8mb4_et_0900_as_cs", false, 1, PadNone},
	{286, "utf8mb4", "utf8mb4_es_0900_as_cs", false, 1, PadNone},
	{287, "utf8mb4", "utf8mb4_sv_0900_as_cs", false, 1, PadNone},
	{288, "utf8mb4", "utf8mb4_tr_0900_as_cs", false, 1, PadNone},
	{289, "utf8mb4", "utf8mb4_cs_0900_as_cs", false, 1, PadNone},
	{290, "utf8mb4", "utf8mb4_da_0900_as_cs", false, 1, PadNone},
	{291, "utf8mb4", "utf8mb4_lt_0900_as_cs", false, 1, PadNone},
	{292, "utf8mb4", "utf8mb4_sk_0900_as_cs", false, 1, PadNone},
	{293, "utf8mb4", "utf8mb4_es_trad_0900_as_cs", false, 1, PadNone},
	{294, "utf8mb4", "utf8mb4_la_0900_as_cs", false, 1, PadNone},
	{296, "utf8mb4", "utf8mb4_eo_0900_as_cs", false, 1, PadNone},
	{297, "utf8mb4", "utf8mb4_hu_0900_as_cs", false, 1, PadNone},
	{298, "utf8mb4", "utf8mb4_hr_0900_as_cs", false, 1, PadNone},
	{300, "utf8mb4", "utf8mb4_vi_0900_as_cs", false, 1, PadNone},
	{303, "utf8mb4", "utf8mb4_ja_0900_as_cs", false, 1, PadNone},
	{304, "utf8mb4", "utf8mb4_ja_0900_as_cs_ks", false, 1, PadNone},
	{305, "utf8mb4", "utf8mb4_0900_as_ci", false, 1, PadNone},
	{306, "utf8mb4", "utf8mb4_ru_0900_ai_ci", false, 1, PadNone},
	{307, "utf8mb4", "utf8mb4_ru_0900_as_cs", false, 1, PadNone},
	{308, "utf8mb4", "utf8mb4_zh_0900_as_cs", false, 1, PadNone},
	{309, "utf8mb4", "utf8mb4_0900_bin", false, 1, PadNone},
	{2048, "utf8mb4", "utf8mb4_zh_pinyin_tidb_as_cs", false, 1, PadNone},
}

// AddCharset adds a new charset.
// Use only when adding a custom charset to the parser.
func AddCharset(c *Charset) {
	CharacterSetInfos[c.Name] = c
}

// RemoveCharset remove a charset.
// Use only when remove a custom charset to the parser.
func RemoveCharset(c string) {
	delete(CharacterSetInfos, c)
	for i := range supportedCollations {
		if supportedCollations[i].Name == c {
			supportedCollations = slices.Delete(supportedCollations, i, i+1)
		}
	}
}

// AddCollation adds a new collation.
// Use only when adding a custom collation to the parser.
func AddCollation(c *Collation) {
	collationsIDMap[c.ID] = c
	collationsNameMap[c.Name] = c

	if _, ok := supportedCollationNames[c.Name]; ok {
		AddSupportedCollation(c)
	}

	if charset, ok := CharacterSetInfos[c.CharsetName]; ok {
		charset.Collations[c.Name] = c
	}

	if charset, ok := charsets[c.CharsetName]; ok {
		charset.Collations[c.Name] = c
	}
}

// AddSupportedCollation adds a new collation into supportedCollations.
// Use only when adding a custom collation to the parser.
func AddSupportedCollation(c *Collation) {
	supportedCollations = append(supportedCollations, c)
}

// init method always puts to the end of file.
func init() {
	for _, c := range collations {
		AddCollation(c)
	}
}
