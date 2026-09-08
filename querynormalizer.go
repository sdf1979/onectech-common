package onectechcommon

import (
	"strings"

	"github.com/DataDog/go-sqllexer"
)

type QueryNormalizer struct {
	obfuscator  *sqllexer.Obfuscator
	normalaizer *sqllexer.Normalizer
}

func NewQueryNormalizer() *QueryNormalizer {
	return &QueryNormalizer{
		obfuscator: sqllexer.NewObfuscator(
			sqllexer.WithReplaceDigits(false),
			sqllexer.WithReplacePositionalParameter(true),
			sqllexer.WithReplaceBindParameter(true),
			sqllexer.WithReplaceNull(true),
		),
		normalaizer: sqllexer.NewNormalizer(
			sqllexer.WithCollectTables(true),
			sqllexer.WithRemoveSpaceBetweenParentheses(true),
		),
	}
}

func (q *QueryNormalizer) Digest(data string, typeDb string) string {
	var digest string
	var err error
	switch typeDb {
	case DBMSSQL:
		data = ReplacePrefixDigits(TrimQuotedString(data), "#tt")
		data, _, _ = strings.Cut(data, "\np_0:")
		digest, _, err = sqllexer.ObfuscateAndNormalize(data, q.obfuscator, q.normalaizer, sqllexer.WithDBMS(sqllexer.DBMSSQLServer))
	case DBPOSTGRS:
		data = ReplacePrefixDigits(TrimQuotedString(data), "pg_temp.tt")
		digest, _, err = sqllexer.ObfuscateAndNormalize(data, q.obfuscator, q.normalaizer, sqllexer.WithDBMS(sqllexer.DBMSPostgres))
	}
	if err != nil {
		return "ERROR DIGEST"
	}
	return digest
}
