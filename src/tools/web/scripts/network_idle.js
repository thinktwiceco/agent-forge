(function() {
    if (window.__agentNetworkPatch) return { pending: window.__agentPending };
    window.__agentPending = 0;
    window.__agentNetworkPatch = true;

    const origFetch = window.fetch;
    window.fetch = function(...args) {
        window.__agentPending++;
        return origFetch.apply(this, args).finally(function() { window.__agentPending--; });
    };

    const origOpen = XMLHttpRequest.prototype.open;
    XMLHttpRequest.prototype.open = function(...args) {
        this.addEventListener('loadend', function() { window.__agentPending--; });
        window.__agentPending++;
        return origOpen.apply(this, args);
    };

    return { pending: window.__agentPending };
})();
