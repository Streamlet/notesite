# KaTeX

Vendored from the npm package `katex@0.16.22` and embedded into the notesite
binary.

Only the woff2 web fonts are shipped; the `@font-face` rules in
`katex.min.css` have been trimmed to reference woff2 only.

`math.css` and `math.js` are notesite's small wrapper: they align display
blocks and call `katex.render` for `.math-inline` / `.math-display` elements.
