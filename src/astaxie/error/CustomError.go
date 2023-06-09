package main

type SyntaxError struct {
	msg    string
	Offset int64
}

func (e *SyntaxError) Error() string {
	return e.msg
}

func main() {
	//	val := `
	//    {"Name": "Ed", "Text": "Knock knock."}
	//    {"Name": "Sam", "Text": "Who's there?"}
	//    {"Name": "Ed", "Text": "Go fmt."}
	//    {"Name": "Sam", "Text": "Go fmt who?"}
	//    {"Name": "Ed", "Text": "Go fmt yourself!"}
	//`
	//	type Message struct {
	//		Name, Text string
	//	}
	//
	//	dec := json.NewDecoder(strings.NewReader(val))
	//
	//	if err := dec.Decode(&val); err != nil {
	//		if serr, ok := err.(*json.SyntaxError); ok {
	//			line, col := findLine(f, serr.Offset)
	//			return fmt.Errorf("%s:%d:%d: %v", f.Name(), line, col, err)
	//		}
	//		return err
	//	}
}
