(function() {
	const interactives = document.querySelectorAll('a, button, input, select, textarea, [role="button"], [role="link"], [tabindex="0"]');
	let results = [];
	interactives.forEach((el, index) => {
		// Skip hidden elements by checking bounding box
		const rect = el.getBoundingClientRect();
		if (rect.width === 0 || rect.height === 0) return;
		
		let type = el.tagName.toLowerCase();
		let role = el.getAttribute('role');
		if (role) type = role;
		
		let identity = [];
		if (el.name) identity.push('name="' + el.name + '"');
		if (el.type) identity.push('type="' + el.type + '"');
		if (el.placeholder) identity.push('placeholder="' + el.placeholder + '"');
		if (el.getAttribute('aria-label')) identity.push('aria-label="' + el.getAttribute('aria-label') + '"');
		
		let text = (el.innerText || el.value || '').trim().replace(/\n/g, ' ');
		if (text && text.length < 100) identity.push('text="' + text + '"'); // Truncate very long text
		
		if (identity.length > 0) {
			results.push('[' + type + '] ' + identity.join(' '));
		}
	});
	return results.join('\n');
})();
