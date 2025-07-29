package sexy

import (
	"fmt"
	"strings"
	"unicode"
)

// NodeType represents the type of a Node
type NodeType int

const (
	NodeSymbol NodeType = iota
	NodeString
	NodeInteger
	NodeEllipsis
	NodeList
	NodeMap
	NodeSet
	NodeArray
	NodeLabelRef
)

// Node represents any Sexy data structure
type Node struct {
	Type NodeType

	// Fields for different node types
	// Atoms and text
	Text string // NodeSymbol, NodeString, NodeInteger, NodeLabelRef

	// Collections
	Items []*Node  // NodeList, NodeSet, NodeArray
	Keys  []string // NodeMap - parallel to Items

	// Metadata for NodeList - stored as parallel slices like maps
	MetaKeys  []string // NodeList - metadata keys
	MetaItems []*Node  // NodeList - metadata values

	// Labels - any node can have a label
	Label string // Label name (empty for unlabeled nodes)
}

func (n *Node) String() string {
	// Get the base string representation
	var baseStr string
	switch n.Type {
	case NodeSymbol:
		baseStr = n.Text
	case NodeString:
		escaped := strings.ReplaceAll(n.Text, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		baseStr = fmt.Sprintf("\"%s\"", escaped)
	case NodeInteger:
		baseStr = n.Text
	case NodeEllipsis:
		baseStr = "..."
	case NodeList:
		var parts []string
		// Add metadata at the beginning if present
		if len(n.MetaKeys) > 0 {
			var metaParts []string
			for i, key := range n.MetaKeys {
				if i < len(n.MetaItems) {
					metaParts = append(metaParts, fmt.Sprintf("%s: %s", key, n.MetaItems[i].String()))
				}
			}
			if len(metaParts) > 0 {
				parts = append(parts, fmt.Sprintf("^{%s}", strings.Join(metaParts, ", ")))
			}
		}
		for _, item := range n.Items {
			parts = append(parts, item.String())
		}
		baseStr = fmt.Sprintf("(%s)", strings.Join(parts, " "))
	case NodeMap:
		var parts []string
		for i, key := range n.Keys {
			if i < len(n.Items) {
				parts = append(parts, fmt.Sprintf("%s: %s", key, n.Items[i].String()))
			}
		}
		baseStr = fmt.Sprintf("{%s}", strings.Join(parts, ", "))
	case NodeSet:
		var parts []string
		for _, item := range n.Items {
			parts = append(parts, item.String())
		}
		baseStr = fmt.Sprintf("{%s}", strings.Join(parts, " "))
	case NodeArray:
		var parts []string
		for _, item := range n.Items {
			parts = append(parts, item.String())
		}
		baseStr = fmt.Sprintf("[%s]", strings.Join(parts, " "))
	case NodeLabelRef:
		baseStr = fmt.Sprintf("#%s#", n.Text)
	default:
		baseStr = fmt.Sprintf("UNKNOWN_NODE_TYPE_%d", n.Type)
	}

	// Add label if present
	if n.Label != "" {
		return fmt.Sprintf("#%s=%s", n.Label, baseStr)
	}
	return baseStr
}

// Helper constructors for common node types
func NewSymbol(name string) *Node {
	return &Node{Type: NodeSymbol, Text: name}
}

func NewString(value string) *Node {
	return &Node{Type: NodeString, Text: value}
}

func NewInteger(text string) *Node {
	return &Node{Type: NodeInteger, Text: text}
}

func NewEllipsis() *Node {
	return &Node{Type: NodeEllipsis}
}

func NewList(items []*Node) *Node {
	return &Node{Type: NodeList, Items: items}
}

func NewListWithMeta(items []*Node, metaKeys []string, metaItems []*Node) *Node {
	return &Node{Type: NodeList, Items: items, MetaKeys: metaKeys, MetaItems: metaItems}
}

func NewMap(keys []string, items []*Node) *Node {
	return &Node{Type: NodeMap, Keys: keys, Items: items}
}

func NewSet(items []*Node) *Node {
	return &Node{Type: NodeSet, Items: items}
}

func NewArray(items []*Node) *Node {
	return &Node{Type: NodeArray, Items: items}
}

func NewLabelRef(name string) *Node {
	return &Node{Type: NodeLabelRef, Text: name}
}

// SetLabel sets the label on a node and returns the node for chaining
func (n *Node) SetLabel(label string) *Node {
	n.Label = label
	return n
}

// IsAtom checks if the node is an atomic value
func (n *Node) IsAtom() bool {
	return n.Type == NodeSymbol || n.Type == NodeString || n.Type == NodeInteger || n.Type == NodeEllipsis
}

// IsLabeled checks if the node has a label
func (n *Node) IsLabeled() bool {
	return n.Label != ""
}

type parser struct {
	lexer        *lexer
	currentToken token
	peekToken    token
}

// Parse parses the entire input and returns the top-level datum
func Parse(input string) (*Node, error) {
	p := &parser{lexer: newLexer(input)}
	p.nextToken()
	p.nextToken()

	result, err := p.ParseDatum()
	if len(p.lexer.errors) > 0 {
		// Lexer errors take priority because they might cause confusing parser errors.
		return nil, fmt.Errorf("%s", p.lexer.errors[0])
	}
	if err != nil {
		return nil, err
	}

	if p.currentToken.Type != tokenEOF {
		// Check for lexer errors first if we didn't get EOF
		return nil, fmt.Errorf("expected EOF but got %s", p.currentToken.Type)
	}

	return result, nil
}

func (p *parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.lexer.nextToken()
}

func (p *parser) ParseDatum() (*Node, error) {
	// Check for label definition
	if p.currentToken.Type == tokenLabelDef {
		labelName := p.currentToken.Value
		p.nextToken()

		data, err := p.parseUnlabeledDatum()
		if err != nil {
			return nil, err
		}

		// Set the label on the node
		data.Label = labelName
		return data, nil
	}

	return p.parseUnlabeledDatum()
}

func (p *parser) parseUnlabeledDatum() (*Node, error) {
	switch p.currentToken.Type {
	case tokenSymbol:
		return p.parseSymbol()
	case tokenString:
		return p.parseString()
	case tokenInteger:
		return p.parseInteger()
	case tokenEllipsis:
		return p.parseEllipsis()
	case tokenLParen:
		return p.parseList()
	case tokenLBrace:
		return p.parseMapOrSet()
	case tokenLBracket:
		return p.parseArray()
	case tokenLabelRef:
		return p.parseLabelRef()
	default:
		return nil, fmt.Errorf("unexpected token: %s", p.currentToken.Type)
	}
}

func (p *parser) parseSymbol() (*Node, error) {
	symbol := NewSymbol(p.currentToken.Value)
	p.nextToken()
	return symbol, nil
}

func (p *parser) parseString() (*Node, error) {
	str := NewString(p.currentToken.Value)
	p.nextToken()
	return str, nil
}

func (p *parser) parseInteger() (*Node, error) {
	text := p.currentToken.Value
	// We could validate it's a valid integer here, but let callers deal with parsing if needed
	integer := NewInteger(text)
	p.nextToken()
	return integer, nil
}

func (p *parser) parseEllipsis() (*Node, error) {
	ellipsis := NewEllipsis()
	p.nextToken()
	return ellipsis, nil
}

func (p *parser) parseLabelRef() (*Node, error) {
	labelRef := NewLabelRef(p.currentToken.Value)
	p.nextToken()
	return labelRef, nil
}

func (p *parser) parseList() (*Node, error) {
	var items []*Node
	var metaKeys []string
	var metaItems []*Node
	p.nextToken() // consume '('

	for p.currentToken.Type != tokenRParen && p.currentToken.Type != tokenEOF {
		if p.currentToken.Type == tokenCaret {
			// Parse metadata and merge it
			metaNode, err := p.parseMeta()
			if err != nil {
				return nil, err
			}

			// Merge metadata keys and items
			if metaNode.Type == NodeMap {
				for i, key := range metaNode.Keys {
					if i < len(metaNode.Items) {
						// Check if key already exists - later values win
						found := false
						for j, existingKey := range metaKeys {
							if existingKey == key {
								metaItems[j] = metaNode.Items[i] // Replace existing value
								found = true
								break
							}
						}
						if !found {
							metaKeys = append(metaKeys, key)
							metaItems = append(metaItems, metaNode.Items[i])
						}
					}
				}
			}
		} else {
			// Parse regular item
			item, err := p.ParseDatum()
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}

	if p.currentToken.Type != tokenRParen {
		return nil, fmt.Errorf("expected ')' but got %s", p.currentToken.Type)
	}
	p.nextToken() // consume ')'

	if len(metaKeys) > 0 {
		return NewListWithMeta(items, metaKeys, metaItems), nil
	}
	return NewList(items), nil
}

func (p *parser) parseMeta() (*Node, error) {
	p.nextToken() // consume '^'

	if p.currentToken.Type != tokenLBrace {
		return nil, fmt.Errorf("expected '{' after '^' but got %s", p.currentToken.Type)
	}

	// Parse the map directly - this will be the metadata node
	return p.parseMap()
}

func (p *parser) parseMapOrSet() (*Node, error) {
	p.nextToken() // consume '{'

	if p.currentToken.Type == tokenRBrace {
		// Empty map
		p.nextToken()
		return NewMap(nil, nil), nil
	}

	// Look ahead to determine if this is a map or set
	// If we see a colon after a symbol, it's a map
	if p.currentToken.Type == tokenSymbol && p.peekToken.Type == tokenColon {
		return p.parseMapFromOpenBrace()
	}

	// Otherwise it's a set
	return p.parseSetFromOpenBrace()
}

func (p *parser) parseMap() (*Node, error) {
	p.nextToken() // consume '{'
	return p.parseMapFromOpenBrace()
}

func (p *parser) parseMapFromOpenBrace() (*Node, error) {
	var keys []string
	var items []*Node

	for p.currentToken.Type != tokenRBrace && p.currentToken.Type != tokenEOF {
		// Parse key (must be symbol)
		if p.currentToken.Type != tokenSymbol {
			return nil, fmt.Errorf("expected symbol for map key but got %s", p.currentToken.Type)
		}

		key := p.currentToken.Value
		keys = append(keys, key)
		p.nextToken()

		// Expect colon
		if p.currentToken.Type != tokenColon {
			return nil, fmt.Errorf("expected ':' after map key but got %s", p.currentToken.Type)
		}
		p.nextToken()

		// Parse value
		value, err := p.ParseDatum()
		if err != nil {
			return nil, err
		}

		items = append(items, value)

		// Check for comma or end
		if p.currentToken.Type == tokenComma {
			p.nextToken()
		} else if p.currentToken.Type != tokenRBrace {
			return nil, fmt.Errorf("expected ',' or '}' in map but got %s", p.currentToken.Type)
		}
	}

	if p.currentToken.Type != tokenRBrace {
		return nil, fmt.Errorf("expected '}' but got %s", p.currentToken.Type)
	}
	p.nextToken() // consume '}'

	return NewMap(keys, items), nil
}

func (p *parser) parseSetFromOpenBrace() (*Node, error) {
	var items []*Node

	for p.currentToken.Type != tokenRBrace && p.currentToken.Type != tokenEOF {
		item, err := p.ParseDatum()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if p.currentToken.Type != tokenRBrace {
		return nil, fmt.Errorf("expected '}' but got %s", p.currentToken.Type)
	}
	p.nextToken() // consume '}'

	return NewSet(items), nil
}

func (p *parser) parseArray() (*Node, error) {
	var items []*Node
	p.nextToken() // consume '['

	for p.currentToken.Type != tokenRBracket && p.currentToken.Type != tokenEOF {
		item, err := p.ParseDatum()
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	if p.currentToken.Type != tokenRBracket {
		return nil, fmt.Errorf("expected ']' but got %s", p.currentToken.Type)
	}
	p.nextToken() // consume ']'

	return NewArray(items), nil
}

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenSymbol
	tokenString
	tokenInteger
	tokenEllipsis
	tokenLParen
	tokenRParen
	tokenLBrace
	tokenRBrace
	tokenLBracket
	tokenRBracket
	tokenColon
	tokenComma
	tokenCaret
	tokenLabelDef
	tokenLabelRef
)

func (t tokenType) String() string {
	switch t {
	case tokenEOF:
		return "EOF"
	case tokenSymbol:
		return "symbol"
	case tokenString:
		return "string"
	case tokenInteger:
		return "integer"
	case tokenEllipsis:
		return "ellipsis"
	case tokenLParen:
		return "'('"
	case tokenRParen:
		return "')'"
	case tokenLBrace:
		return "'{'"
	case tokenRBrace:
		return "'}'"
	case tokenLBracket:
		return "'['"
	case tokenRBracket:
		return "']'"
	case tokenColon:
		return "':'"
	case tokenComma:
		return "','"
	case tokenCaret:
		return "'^'"
	case tokenLabelDef:
		return "label definition"
	case tokenLabelRef:
		return "label reference"
	default:
		return fmt.Sprintf("unknown token %d", int(t))
	}
}

type token struct {
	Type     tokenType
	Value    string
	Position int
}

type lexer struct {
	input    string
	position int
	current  rune
	errors   []string
}

func newLexer(input string) *lexer {
	l := &lexer{input: input}
	l.readChar()
	return l
}

func (l *lexer) readChar() {
	if l.position >= len(l.input) {
		l.current = 0
	} else {
		l.current = rune(l.input[l.position])
	}
	l.position++
}

func (l *lexer) peekChar() rune {
	if l.position >= len(l.input) {
		return 0
	}
	return rune(l.input[l.position])
}

func (l *lexer) skipWhitespace() {
	for unicode.IsSpace(l.current) {
		l.readChar()
	}
}

func (l *lexer) skipComment() {
	for l.current != '\n' && l.current != '\r' && l.current != 0 {
		l.readChar()
	}
}

func (l *lexer) readSymbol() string {
	start := l.position - 1
	for isSymbolChar(l.current) {
		l.readChar()
	}
	return l.input[start : l.position-1]
}

func (l *lexer) readString() (string, error) {
	var result string
	l.readChar() // skip opening quote

	for l.current != '"' && l.current != 0 {
		if l.current == '\\' {
			l.readChar()
			switch l.current {
			case '"':
				result += "\""
			case '\\':
				result += "\\"
			default:
				return "", fmt.Errorf("invalid escape sequence: \\%c", l.current)
			}
		} else {
			result += string(l.current)
		}
		l.readChar()
	}

	if l.current != '"' {
		return "", fmt.Errorf("unterminated string")
	}
	l.readChar() // skip closing quote

	return result, nil
}

func (l *lexer) readInteger() string {
	start := l.position - 1
	if l.current == '+' || l.current == '-' {
		l.readChar()
	}
	for unicode.IsDigit(l.current) {
		l.readChar()
	}
	return l.input[start : l.position-1]
}

func (l *lexer) readLabelName() string {
	start := l.position - 1
	if unicode.IsDigit(l.current) {
		for unicode.IsDigit(l.current) {
			l.readChar()
		}
	} else {
		for isSymbolChar(l.current) {
			l.readChar()
		}
	}
	return l.input[start : l.position-1]
}

func (l *lexer) nextToken() token {
	for {
		l.skipWhitespace()

		pos := l.position - 1

		switch l.current {
		case 0:
			return token{Type: tokenEOF, Position: pos}
		case ';':
			l.skipComment()
			continue
		case '(':
			l.readChar()
			return token{Type: tokenLParen, Value: "(", Position: pos}
		case ')':
			l.readChar()
			return token{Type: tokenRParen, Value: ")", Position: pos}
		case '{':
			l.readChar()
			return token{Type: tokenLBrace, Value: "{", Position: pos}
		case '}':
			l.readChar()
			return token{Type: tokenRBrace, Value: "}", Position: pos}
		case '[':
			l.readChar()
			return token{Type: tokenLBracket, Value: "[", Position: pos}
		case ']':
			l.readChar()
			return token{Type: tokenRBracket, Value: "]", Position: pos}
		case ':':
			l.readChar()
			return token{Type: tokenColon, Value: ":", Position: pos}
		case ',':
			l.readChar()
			return token{Type: tokenComma, Value: ",", Position: pos}
		case '^':
			l.readChar()
			return token{Type: tokenCaret, Value: "^", Position: pos}
		case '#':
			l.readChar()
			labelName := l.readLabelName()
			if l.current == '=' {
				l.readChar()
				return token{Type: tokenLabelDef, Value: labelName, Position: pos}
			} else if l.current == '#' {
				l.readChar()
				return token{Type: tokenLabelRef, Value: labelName, Position: pos}
			}
			return token{Type: tokenSymbol, Value: "#" + labelName, Position: pos}
		case '"':
			str, err := l.readString()
			if err != nil {
				return token{Type: tokenEOF, Value: err.Error(), Position: pos}
			}
			return token{Type: tokenString, Value: str, Position: pos}
		case '.':
			if l.peekChar() == '.' {
				l.readChar()
				if l.peekChar() == '.' {
					l.readChar()
					l.readChar()
					return token{Type: tokenEllipsis, Value: "...", Position: pos}
				}
			}
			// Single dot is a syntax error
			l.errors = append(l.errors, fmt.Sprintf("unexpected character '.'"))
			return token{Type: tokenEOF, Position: pos}
		default:
			if unicode.IsLetter(l.current) {
				symbol := l.readSymbol()
				return token{Type: tokenSymbol, Value: symbol, Position: pos}
			} else if unicode.IsDigit(l.current) || l.current == '+' || l.current == '-' {
				if (l.current == '+' || l.current == '-') && !unicode.IsDigit(l.peekChar()) {
					// Single + or - is a symbol
					symbol := l.readSymbol()
					return token{Type: tokenSymbol, Value: symbol, Position: pos}
				}
				integer := l.readInteger()
				return token{Type: tokenInteger, Value: integer, Position: pos}
			} else {
				// Unknown character is a syntax error
				l.errors = append(l.errors, fmt.Sprintf("unexpected character '%c'", l.current))
				return token{Type: tokenEOF, Position: pos}
			}
		}
	}
}

func isSymbolStart(r rune) bool {
	return unicode.IsLetter(r)
}

func isSymbolChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_'
}
