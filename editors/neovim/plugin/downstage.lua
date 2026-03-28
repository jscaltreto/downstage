vim.lsp.config("downstage", {
	cmd = { "downstage", "lsp" },
	filetypes = { "downstage" },
	root_markers = { ".git" },
})
vim.lsp.enable("downstage")
