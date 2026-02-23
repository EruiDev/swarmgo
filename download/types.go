package download

type PieceWork struct {
	index  int
	hash   []byte
	length int
}

type PieceResult struct {
	index int
	data  []byte
	err   error
}
