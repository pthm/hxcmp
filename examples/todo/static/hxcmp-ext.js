// hxcmp-ext.js
// HTMX extension for hxcmp event data injection and toast auto-dismiss.
// Load after htmx.js: <script src="/static/hxcmp-ext.js"></script>

(function() {
    'use strict';

    function init() {
        // Inject event data into HTMX requests triggered by custom events.
        // When a component emits Trigger("event", data), listeners receive
        // the data as request parameters automatically.
        document.body.addEventListener('htmx:configRequest', function(evt) {
            var elt = evt.detail.elt;

            // URL Sync: Read specified params from browser URL and inject into request.
            // Used for React-like shared state via URL query params.
            // data-sync-url="status,sort" syncs specific params
            // data-sync-url="*" syncs all URL params
            var syncUrl = elt.getAttribute('data-sync-url');
            if (syncUrl) {
                var urlParams = new URLSearchParams(window.location.search);
                if (syncUrl === '*') {
                    // Sync all URL params
                    urlParams.forEach(function(value, key) {
                        evt.detail.parameters[key] = value;
                    });
                } else {
                    // Sync only specified params
                    var paramNames = syncUrl.split(',');
                    for (var i = 0; i < paramNames.length; i++) {
                        var name = paramNames[i].trim();
                        if (urlParams.has(name)) {
                            evt.detail.parameters[name] = urlParams.get(name);
                        }
                    }
                }
            }

            // Event data injection (existing behavior)
            var triggeringEvent = evt.detail.triggeringEvent;
            if (!triggeringEvent || !triggeringEvent.detail) {
                return;
            }

            var data = triggeringEvent.detail;
            if (typeof data !== 'object' || data === null) {
                return;
            }

            // Inject each key from the event detail into request parameters
            for (var key in data) {
                if (data.hasOwnProperty(key)) {
                    evt.detail.parameters[key] = data[key];
                }
            }
        });

        // [Deprecated] Listen for hxcmp:callback events.
        // Callbacks are deprecated in favor of Trigger with data.
        document.body.addEventListener('hxcmp:callback', function(evt) {
            var detail = evt.detail || {};
            var data = detail.value || detail;

            if (!data.url) {
                console.warn('hxcmp:callback event missing url');
                return;
            }

            var url = data.url;
            if (data.vals && typeof data.vals === 'object') {
                var params = new URLSearchParams();
                for (var key in data.vals) {
                    if (data.vals.hasOwnProperty(key)) {
                        params.append(key, data.vals[key]);
                    }
                }
                var queryString = params.toString();
                if (queryString) {
                    url += (url.indexOf('?') === -1 ? '?' : '&') + queryString;
                }
            }

            htmx.ajax('GET', url, {
                target: data.target || 'body',
                swap: data.swap || 'outerHTML'
            });
        });

        // URL Sync: Trigger url:sync on browser back/forward navigation.
        // This enables components with SyncURL() to re-render when user navigates history.
        window.addEventListener('popstate', function() {
            htmx.trigger(document.body, 'url:sync');
        });

        // Auto-dismiss toasts after their configured delay.
        document.body.addEventListener('htmx:afterSwap', function(evt) {
            var toasts = evt.detail.target.querySelectorAll('[data-auto-dismiss]');
            toasts.forEach(function(toast) {
                var delay = parseInt(toast.getAttribute('data-auto-dismiss'), 10) || 3000;
                setTimeout(function() {
                    toast.classList.add('toast-fade-out');
                    setTimeout(function() {
                        toast.remove();
                    }, 300);
                }, delay);
            });
        });

        console.log('hxcmp extension loaded');
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
