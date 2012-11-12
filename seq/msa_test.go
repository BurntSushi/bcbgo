package seq

import (
	"fmt"
	"testing"
)

func TestMSAA3M(t *testing.T) {
	tests := [][]string{
		alignA3M,
	}
	answers := [][]string{
		alignA2M,
	}
	for i := 0; i < len(tests); i++ {
		test := makeSeqs(tests[i])
		answer := makeMSA(makeSeqs(answers[i]))

		computed := NewMSA()
		computed.AddSlice(test)

		testEqualAlign(t, computed, answer)
	}
}

func TestMSAA2M(t *testing.T) {
	tests := [][]string{
		alignA2M,
	}
	answers := [][]string{
		alignA2M,
	}
	for i := 0; i < len(tests); i++ {
		test := makeSeqs(tests[i])
		answer := makeMSA(makeSeqs(answers[i]))

		computed := NewMSA()
		computed.AddSlice(test)

		testEqualAlign(t, computed, answer)
	}
}

func TestMSAFasta(t *testing.T) {
	tests := [][]string{
		alignFasta,
	}
	answers := [][]string{
		alignA2M,
	}
	for i := 0; i < len(tests); i++ {
		test := makeSeqs(tests[i])
		answer := makeMSA(makeSeqs(answers[i]))

		computed := NewMSA()
		computed.AddSlice(test)

		testEqualAlign(t, computed, answer)
	}
}

func TestGetA3M(t *testing.T) {
	test := "ABCD---...ABCD"
	answer := []Residue("ABCD---ABCD")
	msa := makeMSA(makeSeqs([]string{test}))
	testEqualSeq(t, msa.GetA3M(0).Residues, answer)
}

func TestGetA2M(t *testing.T) {
	test := "ABCD---...ABCD"
	msa := makeMSA(makeSeqs([]string{test}))
	testEqualSeq(t, msa.GetA2M(0).Residues, []Residue(test))
}

func TestGetFasta(t *testing.T) {
	test := "ABCD---...ABCD"
	answer := []Residue("ABCD------ABCD")
	msa := makeMSA(makeSeqs([]string{test}))
	testEqualSeq(t, msa.GetFasta(0).Residues, answer)
}

func testEqualAlign(t *testing.T, computed, answer MSA) {
	if computed.Len() != answer.Len() {
		t.Fatalf("Lengths of MSAs differ: %d != %d",
			computed.Len(), answer.Len())
	}

	scomputed := makeStrings(computed.Entries)
	sanswer := makeStrings(answer.Entries)
	if len(scomputed) != len(sanswer) {
		t.Fatalf("\nLengths of entries in MSAs differ: %d != %d",
			len(scomputed), len(sanswer))
	}
	for i := 0; i < len(scomputed); i++ {
		c, a := scomputed[i], sanswer[i]
		if c != a {
			t.Fatalf("\nComputed sequence in MSA is\n\n%s\n\n"+
				"but answer is\n\n%s", c, a)
		}
	}
}

func testEqualSeq(t *testing.T, computed, answer []Residue) {
	scomputed := fmt.Sprintf("%s", computed)
	sanswer := fmt.Sprintf("%s", answer)
	if scomputed != sanswer {
		t.Fatalf("\nComputed sequence is\n\n%s\n\n"+
			"but answer is\n\n%s", scomputed, sanswer)
	}
}

func makeMSA(seqs []Sequence) MSA {
	return MSA{
		Entries: seqs,
		length:  len(seqs[0].Residues),
	}
}

func makeSeqs(strs []string) []Sequence {
	seqs := make([]Sequence, len(strs))
	for i, str := range strs {
		seqs[i] = Sequence{
			Name:     fmt.Sprintf("%d", i),
			Residues: []Residue(str),
		}
	}
	return seqs
}

func makeStrings(seqs []Sequence) []string {
	strs := make([]string, len(seqs))
	for i, s := range seqs {
		strs[i] = fmt.Sprintf("%s", s.Residues)
	}
	return strs
}
