package parser

type Token int

const EOF = 0

const (
	tokenNumber     = NUMBER      // integer literal
	tokenIdentifier = IDENTIFIER  // programmer defined names
	tokenSeparator  = SEPARATOR   // separator e.g. semicolon
	tokenAssign     = ASSIGN      // assignment operator
	tokenLet        = LET         // let keyword
	tokenIf         = IF          // if keyword
	tokenElse       = ELSE        // else keyword
	tokenGt         = GT          // >
	tokenLt         = LT          // <
	tokenLte        = LTE         // <=
	tokenGte        = GTE         // >=
	tokenEq         = EQ          // ==
	tokenNe         = NE          // !=
	tokenOr         = OR          // ||
	tokenAnd        = AND         // &&
	tokenWhile      = WHILE       // while keyword
	tokenPrint      = PRINT       // print keyword
	tokenFunc       = FUNC        // func keyword
	tokenReturn     = RETURN      // return keyword
	tokenTypeInt    = TYPE_INT    // int type name
	tokenTypeDouble = TYPE_DOUBLE // double type name
	tokenTypeBool   = TYPE_BOOL   // bool type name
	tokenTypeVoid   = TYPE_VOID   // void type name
	tokenTrue       = TRUE        // boolean literal true
	tokenFalse      = FALSE       // boolean literal false
)
