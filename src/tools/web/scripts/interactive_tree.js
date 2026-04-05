(function() {
    const selector = [
        'a[href]', 'button', 'input', 'select', 'textarea',
        '[role="button"]', '[role="link"]', '[role="tab"]',
        '[role="menuitem"]', '[role="menuitemcheckbox"]', '[role="menuitemradio"]',
        '[role="option"]', '[role="combobox"]', '[role="listbox"]',
        '[role="switch"]', '[role="checkbox"]', '[role="radio"]',
        '[role="slider"]', '[role="spinbutton"]', '[role="searchbox"]',
        '[role="textbox"]', '[role="gridcell"][aria-selected]',
        '[tabindex="0"]', '[aria-haspopup]', '[data-testid]',
    ].join(', ');

    const interactives = document.querySelectorAll(selector);
    const results = [];

    interactives.forEach(function(el) {
        const rect = el.getBoundingClientRect();
        if (rect.width === 0 || rect.height === 0) return;

        const type = el.getAttribute('role') || el.tagName.toLowerCase();
        const attrs = [];

        ['name', 'type', 'placeholder', 'aria-label', 'aria-description',
            'data-testid', 'href'].forEach(function(attr) {
            const v = el.getAttribute(attr);
            if (v) attrs.push(attr + '="' + v + '"');
        });

        ['aria-pressed', 'aria-checked', 'aria-expanded',
            'aria-selected', 'aria-disabled'].forEach(function(attr) {
            const v = el.getAttribute(attr);
            if (v !== null) attrs.push(attr + '=' + v);
        });

        const text = (el.innerText || el.value || '').trim()
            .replace(/\s+/g, ' ').slice(0, 80);
        if (text) attrs.push('text="' + text + '"');

        if (attrs.length > 0) {
            results.push('[' + type + '] ' + attrs.join(' '));
        }
    });

    return results.join('\n');
})();
