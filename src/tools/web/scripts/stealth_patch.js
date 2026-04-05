Object.defineProperty(navigator, 'webdriver', { get: function() { return false; } });

Object.defineProperty(navigator, 'plugins', {
    get: function() {
        return [{ name: 'Chrome PDF Plugin' }, { name: 'Chrome PDF Viewer' }];
    }
});

Object.defineProperty(navigator, 'languages', { get: function() { return ['en-US', 'en']; } });

window.chrome = { runtime: {} };
