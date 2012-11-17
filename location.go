package sexp

// Compressed SourceLocEx, can be decoded using an appropriate SourceContext.
type SourceLoc uint32

// Complete source location information. Line number starts from 1, it is a
// traditional choice. The column is specified in bytes, because it makes it
// character encoding agnostic. You can always get column in characters by
// counting them in the source file between Offset-Column and Offset bytes.
// And byte offset is here for the purpose stated above.
type SourceLocEx struct {
	Filename string
	Line     int // starting from 1
	Column   int // starting from 0 (in bytes)
	Offset   int // (in bytes)
}

type source_line struct {
	offset int // relative to the beginning of the file
	num    int // line number
}

// Represents one file within source context, usually a parser will require
// you to pass source file before parsing. Parser should use SourceFile.Encode
// method to encode source location information, method takes byte offset from
// the beginning of the file as an argument.
type SourceFile struct {
	name   string
	offset SourceLoc // relative to the beginning of the SourceContext
	length int
	lines  []source_line
}

// Returns the last line in the file, assumes there is at least one line.
// Which is usually true, since one line is automatically added by the
// SourceContext.AddFile.
func (f *SourceFile) last_line() source_line {
	return f.lines[len(f.lines)-1]
}

// Find line for a given file offset.
func (f *SourceFile) find_line(offset int) source_line {
	// simple binary search, we know that lines are sorted
	beg, end := 0, len(f.lines)
	for {
		len := end - beg
		if len == 1 {
			return f.lines[beg]
		}
		mid := beg + len/2
		if f.lines[mid].offset > offset {
			end = mid
			continue
		} else {
			beg = mid
			continue
		}
	}
	panic("unreachable")
}

// Adds a new line with a given offset, keep in mind that the first line is added
// automatically by SourceContext.AddFile. A parser typically calls that method
// each time it encounters a newline character.
func (f *SourceFile) AddLine(offset int) {
	f.lines = append(f.lines, source_line{
		offset: offset,
		num:    f.last_line().num + 1,
	})
}

// Encodes an offset from the beginning of the file as a source location.
func (f *SourceFile) Encode(offset int) SourceLoc {
	return f.offset + SourceLoc(offset)
}

// If the length of the file is unknown at the beginning, the file must be
// finalized at some point using this method. Otherwise no new files can be
// added to the source context.
func (f *SourceFile) Finalize(len int) {
	f.length = len
}

// Source context holds information needed to decompress source locations.
// It supports multiple files with knowns and unknowns lengths. Although
// having a file with unknown length prevents you from adding more files
// until it's been finalized.
type SourceContext struct {
	files []*SourceFile
}

// Returns the last file in the context, assumes there is at least one file.
func (s *SourceContext) last_file() *SourceFile {
	return s.files[len(s.files)-1]
}

// Find file for a given source location.
func (s *SourceContext) find_file(l SourceLoc) *SourceFile {
	// simple binary search, we know that files are sorted
	beg, end := 0, len(s.files)
	for {
		len := end - beg
		if len == 1 {
			return s.files[beg]
		}
		mid := beg + len/2
		if s.files[mid].offset > l {
			end = mid
			continue
		} else {
			beg = mid
			continue
		}
	}
	panic("unreachable")
}

// Adds a new file to the context, use -1 as length if the length is unknown, but
// keep in mind that having a file with unknown length prevents further
// AddFile calls, they will panic. In order to continue adding files to the
// context, the last file with unknown length must be finalized. Method doesn't
// read anything, all the arguments are purely informative.
func (s *SourceContext) AddFile(filename string, length int) *SourceFile {
	if len(s.files) != 0 && s.last_file().length == -1 {
		panic("sexp: last file in the SourceContext was not finalized")
	}

	offset := SourceLoc(0)
	if len(s.files) != 0 {
		last := s.last_file()
		offset = last.offset + SourceLoc(last.length)
	}

	f := &SourceFile{
		name:   filename,
		offset: offset,
		length: length,
		lines:  []source_line{{0, 1}},
	}
	s.files = append(s.files, f)
	return f
}

// Decodes an encoded source location.
func (s *SourceContext) Decode(loc SourceLoc) SourceLocEx {
	if len(s.files) == 0 {
		panic("sexp: decoding SourceLoc in a wrong SourceContext")
	}

	file := s.find_file(loc)
	offset := int(loc - file.offset)
	line := file.find_line(offset)
	return SourceLocEx{
		Filename: file.name,
		Line:     line.num,
		Column:   offset - line.offset,
		Offset:   offset,
	}
}
