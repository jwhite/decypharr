package models

type TorboxFile struct {
	Id           int    `json:"id"`
	Md5          any    `json:"md5"`
	Hash         string `json:"hash"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	Zipped       bool   `json:"zipped"`
	S3Path       string `json:"s3_path"`
	Infected     bool   `json:"infected"`
	Mimetype     string `json:"mimetype"`
	ShortName    string `json:"short_name"`
	AbsolutePath string `json:"absolute_path"`
}
