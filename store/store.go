package store

type Store interface {
	//AddURLData(url string, data []byte, result interface{}) error
	//SaveExtractedTextAndLinks(id string, data []byte, result interface{}) error
	//GetURLData(id string, result interface{}) error
	//IsURLThere(target string) bool
	IsItParsed(target string) bool
}
