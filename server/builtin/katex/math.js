(function () {
	if (!window.katex) {
		return;
	}

	document.querySelectorAll(".math-inline, .math-display").forEach(function (element) {
		var display = element.classList.contains("math-display");
		var open = display ? "\\[" : "\\(";
		var close = display ? "\\]" : "\\)";
		var tex = element.textContent;
		if (tex.indexOf(open) === 0 && tex.lastIndexOf(close) === tex.length - close.length) {
			tex = tex.slice(open.length, tex.length - close.length);
		}
		try {
			katex.render(tex, element, { displayMode: display, throwOnError: false });
		} catch (e) {
			// Leave the source text in place if rendering fails.
		}
	});
})();
