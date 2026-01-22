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
