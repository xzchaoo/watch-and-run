package war

import (
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIgnore(t *testing.T) {
	gi := gitignore.CompileIgnoreLines("benchmarks", "*.txt", "target", "*.md", "/aaa", "t2/")

	assert.True(t, gi.MatchesPath("/home/admin/xzchaoo/benchmarks/foo.txt"))
	assert.True(t, gi.MatchesPath("/home/admin/xzchaoo/bench2marks/foo.txt"))
	assert.True(t, gi.MatchesPath("/home/admin/xzchaoo/bench2marks/foo.txt"))
	assert.True(t, gi.MatchesPath("/home/admin/xzchaoo/benchmarks/foo.go"))
	assert.False(t, gi.MatchesPath("/home/admin/xzchaoo/bench2marks/foo.java"))
	assert.True(t, gi.MatchesPath("/home/admin/xzchaoo/target/classes/foo.class"))
	assert.True(t, gi.MatchesPath("/home/admin/xzchaoo/README.md"))

	// 转成相对 root 的路径后扔进去判断就可以命中  /aaa 规则
	assert.True(t, gi.MatchesPath("aaa/README.mdx"))

	assert.True(t, gi.MatchesPath("bbb/t2/README.mdx"))
	assert.False(t, gi.MatchesPath("bbb/t2"))
	assert.True(t, gi.MatchesPath("bbb/t2/"))
}
