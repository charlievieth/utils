package main

import (
	"testing"
)

type parseTest struct {
	In, Want string
	Match    Match
}

var parseTests = []parseTest{
	{
		In:    "\x1b[0m\x1b[35mconfig/plug.vim\x1b[0m:\x1b[0m\x1b[32m19\x1b[0m:\x1b[0m7\x1b[0m:let g:\x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m_command_prefix = '\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0m'",
		Want:  "config/plug.vim:19:7:let g:fzf_command_prefix = 'FZF'",
		Match: Match{Filename: "config/plug.vim", Line: 19, Column: 7},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plug.vim\x1b[0m:\x1b[0m\x1b[32m20\x1b[0m:\x1b[0m16\x1b[0m:Plug 'junegunn/\x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m', { 'dir': '~/.\x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m', 'do': './install --all' }",
		Want:  "config/plug.vim:20:16:Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' }",
		Match: Match{Filename: "config/plug.vim", Line: 20, Column: 16},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plug.vim\x1b[0m:\x1b[0m\x1b[32m21\x1b[0m:\x1b[0m16\x1b[0m:Plug 'junegunn/\x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m.vim'",
		Want:  "config/plug.vim:21:16:Plug 'junegunn/fzf.vim'",
		Match: Match{Filename: "config/plug.vim", Line: 21, Column: 16},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m16\x1b[0m:\x1b[0m8\x1b[0m:  let $\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0m_DEFAULT_COMMAND = '_vim_fd'",
		Want:  "config/plugin/fzf.vim:16:8:  let $FZF_DEFAULT_COMMAND = '_vim_fd'",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 16, Column: 8},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m19\x1b[0m:\x1b[0m8\x1b[0m:  let $\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0m_DEFAULT_COMMAND = 'sfd --type file --follow --hidden --exclude .git --exclude .DS_Store --exclude .cache'",
		Want:  "config/plugin/fzf.vim:19:8:  let $FZF_DEFAULT_COMMAND = 'sfd --type file --follow --hidden --exclude .git --exclude .DS_Store --exclude .cache'",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 19, Column: 8},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m21\x1b[0m:\x1b[0m8\x1b[0m:  let $\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0m_DEFAULT_COMMAND = 'fd --type file --follow --hidden --exclude .git --exclude .DS_Store --exclude .cache'",
		Want:  "config/plugin/fzf.vim:21:8:  let $FZF_DEFAULT_COMMAND = 'fd --type file --follow --hidden --exclude .git --exclude .DS_Store --exclude .cache'",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 21, Column: 8},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m23\x1b[0m:\x1b[0m8\x1b[0m:  let $\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0m_DEFAULT_COMMAND = 'rg --files --hidden --follow --glob \"!.git/*\"",
		Want:  "config/plugin/fzf.vim:23:8:  let $FZF_DEFAULT_COMMAND = 'rg --files --hidden --follow --glob \"!.git/*\"",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 23, Column: 8},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m26\x1b[0m:\x1b[0m8\x1b[0m:  let $\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0m_DEFAULT_COMMAND = 'ag -g \"\"'",
		Want:  "config/plugin/fzf.vim:26:8:  let $FZF_DEFAULT_COMMAND = 'ag -g \"\"'",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 26, Column: 8},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m29\x1b[0m:\x1b[0m25\x1b[0m:command! -bang -nargs=* \x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mRg",
		Want:  "config/plugin/fzf.vim:29:25:command! -bang -nargs=* FZFRg",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 29, Column: 25},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m30\x1b[0m:\x1b[0m14\x1b[0m:      \\ call \x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m#vim#grep(",
		Want:  "config/plugin/fzf.vim:30:14:      \\ call fzf#vim#grep(",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 30, Column: 14},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m36\x1b[0m:\x1b[0m10\x1b[0m:  \\ call \x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m#vim#grep(",
		Want:  "config/plugin/fzf.vim:36:10:  \\ call fzf#vim#grep(",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 36, Column: 10},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m38\x1b[0m:\x1b[0m7\x1b[0m:  \\   \x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m#vim#with_preview(), <bang>0)",
		Want:  "config/plugin/fzf.vim:38:7:  \\   fzf#vim#with_preview(), <bang>0)",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 38, Column: 7},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m40\x1b[0m:\x1b[0m18\x1b[0m:function! Ripgrep\x1b[0m\x1b[1m\x1b[31mFzf\x1b[0m(query, fullscreen)",
		Want:  "config/plugin/fzf.vim:40:18:function! RipgrepFzf(query, fullscreen)",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 40, Column: 18},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m45\x1b[0m:\x1b[0m8\x1b[0m:  call \x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m#vim#grep(initial_command, 1, \x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m#vim#with_preview(spec), a:fullscreen)",
		Want:  "config/plugin/fzf.vim:45:8:  call fzf#vim#grep(initial_command, 1, fzf#vim#with_preview(spec), a:fullscreen)",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 45, Column: 8},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m48\x1b[0m:\x1b[0m40\x1b[0m:command! -nargs=* -bang RG call Ripgrep\x1b[0m\x1b[1m\x1b[31mFzf\x1b[0m(<q-args>, <bang>0)",
		Want:  "config/plugin/fzf.vim:48:40:command! -nargs=* -bang RG call RipgrepFzf(<q-args>, <bang>0)",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 48, Column: 40},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m50\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <C-p>     :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mFiles\n",
		Want:  "config/plugin/fzf.vim:50:21:nnoremap <C-p>     :FZFFiles\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 50, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m51\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>f :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mFiles\n",
		Want:  "config/plugin/fzf.vim:51:21:nnoremap <leader>f :FZFFiles\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 51, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m52\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>m :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mHistory\n",
		Want:  "config/plugin/fzf.vim:52:21:nnoremap <leader>m :FZFHistory\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 52, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m53\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>F :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mBTags\n",
		Want:  "config/plugin/fzf.vim:53:21:nnoremap <leader>F :FZFBTags\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 53, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m54\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>S :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mTags\n",
		Want:  "config/plugin/fzf.vim:54:21:nnoremap <leader>S :FZFTags\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 54, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m55\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>L :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mLines\n",
		Want:  "config/plugin/fzf.vim:55:21:nnoremap <leader>L :FZFLines\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 55, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m56\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>b :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mBuffers\n",
		Want:  "config/plugin/fzf.vim:56:21:nnoremap <leader>b :FZFBuffers\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 56, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m57\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>C :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mColors\n",
		Want:  "config/plugin/fzf.vim:57:21:nnoremap <leader>C :FZFColors\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 57, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m58\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>G :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mRg<space>",
		Want:  "config/plugin/fzf.vim:58:21:nnoremap <leader>G :FZFRg<space>",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 58, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m59\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>: :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mHistory:\n",
		Want:  "config/plugin/fzf.vim:59:21:nnoremap <leader>: :FZFHistory:\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 59, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m60\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>? :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mHistory/\n",
		Want:  "config/plugin/fzf.vim:60:21:nnoremap <leader>? :FZFHistory/\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 60, Column: 21},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m61\x1b[0m:\x1b[0m25\x1b[0m:nnoremap <leader><c-j> :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mSnippets\n",
		Want:  "config/plugin/fzf.vim:61:25:nnoremap <leader><c-j> :FZFSnippets\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 61, Column: 25},
	},
	{
		In:    "\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m62\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>d :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mCommands\n",
		Want:  "config/plugin/fzf.vim:62:21:nnoremap <leader>d :FZFCommands\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 62, Column: 21},
	},
	{
		In:    "config/plugin/fzf.vim:62:21:nnoremap <leader>d :FZFCommands\n",
		Want:  "config/plugin/fzf.vim:62:21:nnoremap <leader>d :FZFCommands\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 62, Column: 21},
	},
	{
		In:    "config/plugin/fzf.vim:62:21::11:22:\n",
		Want:  "config/plugin/fzf.vim:62:21::11:22:\n",
		Match: Match{Filename: "config/plugin/fzf.vim", Line: 62, Column: 21},
	},
	// ":" in filename
	{
		In:    "config/plugin:123/fzf.vim:62:21:11:22:\n",
		Want:  "config/plugin:123/fzf.vim:62:21:11:22:\n",
		Match: Match{Filename: "config/plugin:123/fzf.vim", Line: 62, Column: 21},
	},
	{
		In:    "config/plugin:123:456/fzf.vim:62:21:11:22:\n",
		Want:  "config/plugin:123:456/fzf.vim:62:21:11:22:\n",
		Match: Match{Filename: "config/plugin:123:456/fzf.vim", Line: 62, Column: 21},
	},
	{
		In:    "config/plugin::XX/fzf.vim:62:21:11:22:\n",
		Want:  "config/plugin::XX/fzf.vim:62:21:11:22:\n",
		Match: Match{Filename: "config/plugin::XX/fzf.vim", Line: 62, Column: 21},
	},
}

func init() {
	for i, x := range parseTests {
		parseTests[i].Match.Raw = x.In
	}
}

func TestStripANSI(t *testing.T) {
	tests := append(parseTests,
		parseTest{
			In:   "\x1b[0m",
			Want: "",
		},
		parseTest{
			In:   "Hello World!",
			Want: "Hello World!",
		},
	)
	for i, x := range tests {
		got := stripANSI(x.In)
		if got != x.Want {
			t.Errorf("%d: stripANSI(%q)\n    got:  %q\n    want: %q\n", i, x.In, got, x.Want)
		}
	}
}

func TestParseMatch(t *testing.T) {
	for i, x := range parseTests {
		got, err := parseMatch(x.In)
		if err != nil {
			t.Errorf("%d: %v\n", i, err)
			continue
		}
		if *got != x.Match {
			t.Errorf("%d: parseMatch(%q)\n    got:  %+v\n    want: %+v\n", i, x.In, *got, x.Match)
		}
	}
}

var benchStrings = [...]string{
	"config/plugin/fzf.vim:50:21:nnoremap <C-p>     :FZFFiles\n",
	"config/plugin/fzf.vim:51:21:nnoremap <leader>f :FZFFiles\n",
	"config/plugin/fzf.vim:52:21:nnoremap <leader>m :FZFHistory\n",
	"config/plugin/fzf.vim:53:21:nnoremap <leader>F :FZFBTags\n",
	"config/plug.vim:19:7:let g:fzf_command_prefix = 'FZF'",
	"config/plug.vim:20:16:Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' }",
	"config/plugin/fzf.vim:30:14:      \\ call fzf#vim#grep(",
	"config/plug.vim:21:16:Plug 'junegunn/fzf.vim'",
}

var benchStringsANSI = [...]string{
	"\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m50\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <C-p>     :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mFiles\n",
	"\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m51\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>f :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mFiles\n",
	"\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m52\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>m :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mHistory\n",
	"\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m53\x1b[0m:\x1b[0m21\x1b[0m:nnoremap <leader>F :\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0mBTags\n",
	"\x1b[0m\x1b[35mconfig/plug.vim\x1b[0m:\x1b[0m\x1b[32m19\x1b[0m:\x1b[0m7\x1b[0m:let g:\x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m_command_prefix = '\x1b[0m\x1b[1m\x1b[31mFZF\x1b[0m'",
	"\x1b[0m\x1b[35mconfig/plug.vim\x1b[0m:\x1b[0m\x1b[32m20\x1b[0m:\x1b[0m16\x1b[0m:Plug 'junegunn/\x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m', { 'dir': '~/.\x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m', 'do': './install --all' }",
	"\x1b[0m\x1b[35mconfig/plugin/fzf.vim\x1b[0m:\x1b[0m\x1b[32m30\x1b[0m:\x1b[0m14\x1b[0m:      \\ call \x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m#vim#grep(",
	"\x1b[0m\x1b[35mconfig/plug.vim\x1b[0m:\x1b[0m\x1b[32m21\x1b[0m:\x1b[0m16\x1b[0m:Plug 'junegunn/\x1b[0m\x1b[1m\x1b[31mfzf\x1b[0m.vim'",
}

func BenchmarkStripANSI(b *testing.B) {
	a := &benchStringsANSI
	var n int64
	for i := 0; i < len(a); i++ {
		n += int64(len(a[i]))
	}
	b.SetBytes(n / int64(len(a)))
	for i := 0; i < b.N; i++ {
		stripANSI(a[i%len(a)])
	}
}

func BenchmarkParseMatch(b *testing.B) {
	a := &benchStrings
	var n int64
	for i := 0; i < len(a); i++ {
		n += int64(len(a[i]))
	}
	b.SetBytes(n / int64(len(a)))
	for i := 0; i < b.N; i++ {
		if _, err := parseMatch(a[i%len(a)]); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseMatch_ANSI(b *testing.B) {
	a := &benchStringsANSI
	var n int64
	for i := 0; i < len(a); i++ {
		n += int64(len(a[i]))
	}
	b.SetBytes(n / int64(len(a)))
	for i := 0; i < b.N; i++ {
		if _, err := parseMatch(a[i%len(a)]); err != nil {
			b.Fatal(err)
		}
	}
}
