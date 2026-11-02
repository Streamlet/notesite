(function () {
	function codeText(pre) {
		var code = pre.querySelector("code");
		if (!code) {
			return pre.innerText;
		}
		var lines = [];
		Array.prototype.forEach.call(code.children, function (line) {
			var clone = line.cloneNode(true);
			Array.prototype.forEach.call(clone.querySelectorAll("span"), function (span) {
				if (span.style && span.style.userSelect === "none") {
					span.remove();
				}
			});
			lines.push(clone.textContent);
		});
		return lines.length > 0 ? lines.join("\n") : code.textContent;
	}

	function addCopyButton(pre) {
		if (!navigator.clipboard) {
			return;
		}
		var button = document.createElement("button");
		button.type = "button";
		button.className = "copy-code";
		button.textContent = "Copy";
		button.addEventListener("click", function () {
			navigator.clipboard.writeText(codeText(pre)).then(function () {
				button.textContent = "Copied";
				setTimeout(function () {
					button.textContent = "Copy";
				}, 1500);
			});
		});
		pre.appendChild(button);
	}

	document.querySelectorAll(".doc pre, .quickstart pre").forEach(addCopyButton);
})();
