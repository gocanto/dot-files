" -------------------------------
" Basic settings
" -------------------------------
syntax on                      " Enable syntax highlighting
filetype plugin indent on      " Enable filetype detection, plugins, and smart indentation
set number                     " Show line numbers
set relativenumber             " Show relative line numbers (e.g., for jumping with '5j')
set showcmd                    " Show partial commands in the last line of the screen
set cursorline                 " Highlight the current line
set wildmenu                   " Better command-line completion menu
set title                      " Show file name in terminal title bar

" -------------------------------
" Display & colors
" -------------------------------
set background=dark            " Use colors optimized for dark terminal backgrounds
colorscheme desert             " Set color theme (try: elflord, evening, murphy, torte)

" -------------------------------
" Indentation & tabs
" -------------------------------
set expandtab                  " Use spaces instead of tabs
set tabstop=4                  " Number of spaces a <Tab> displays as
set shiftwidth=4               " Number of spaces used for each step of (auto)indent
set smartindent                " Auto-indent new lines
set autoindent                 " Copy indent from current line when starting a new one

" -------------------------------
" Search
" -------------------------------
set ignorecase                 " Case-insensitive search...
set smartcase                  " ...unless search has uppercase characters
set incsearch                  " Show matches as you type
set hlsearch                   " Highlight search results

" -------------------------------
" Files & backups
" -------------------------------
set nobackup                   " Don't keep backup files
set nowritebackup              " Don't keep backup files before overwriting
set noswapfile                 " Don't create .swp files

" -------------------------------
" Navigation
" -------------------------------
set scrolloff=5                " Keep 5 lines visible when scrolling
set sidescrolloff=5            " Keep 5 columns visible horizontally

" -------------------------------
" Clipboard
" -------------------------------
set clipboard=unnamedplus      " Use system clipboard for yank/paste

" -------------------------------
" Performance
" -------------------------------
set updatetime=300             " Faster completion and cursor hold events

" -------------------------------
" Misc
" -------------------------------
set mouse=a                    " Enable mouse support
set encoding=utf-8             " Ensure UTF-8 encoding

" -------------------------------
" Optional: Startup message
" -------------------------------
" echo "Welcome to Vim 🎉"

