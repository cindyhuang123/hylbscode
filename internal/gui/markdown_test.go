package gui

import (
	"strings"
	"testing"
)

func TestMarkdownBlocksSplitsCodeBlocks(t *testing.T) {
	blocks := markdownBlocks("intro\n```go\nfmt.Println(\"hi\")\n```\noutro")
	if len(blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(blocks))
	}
	if !blocks[0].style.Monospace && !blocks[1].style.Monospace {
		t.Fatal("expected a monospace block for code fence")
	}
	if !strings.Contains(blocks[1].text, "fmt.Println") {
		t.Fatalf("expected code content in block, got %q", blocks[1].text)
	}
}

func TestMarkdownBlocksHeadingBold(t *testing.T) {
	blocks := markdownBlocks("# Title\nplain")
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if !blocks[0].style.Bold {
		t.Fatal("expected heading block bold")
	}
	if blocks[1].style.Bold {
		t.Fatal("expected plain block not bold")
	}
}

func TestMarkdownBlocksEmptyLinesSkipped(t *testing.T) {
	blocks := markdownBlocks("a\n\n\nb")
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if blocks[0].text != "a" || blocks[1].text != "b" {
		t.Fatalf("unexpected blocks: %q, %q", blocks[0].text, blocks[1].text)
	}
}
