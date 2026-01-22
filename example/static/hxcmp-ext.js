// hxcmp-ext.js
// HTMX extension for hxcmp callbacks and toast auto-dismiss.
// Load after htmx.js: <script src="/static/hxcmp-ext.js"></script>

(function() {
    'use strict';

    function init() {
        // Listen for hxcmp:callback events triggered via HX-Trigger header.
        document.body.addEventListener('hxcmp:callback', function(evt) {
            var detail = evt.detail || {};

            // Support both direct detail and nested structure from HX-Trigger.
            var data = detail.value || detail;

            if (!data.url) {
                console.warn('hxcmp:callback event missing url');
                return;
            }

            // Build URL with vals as query params if present.
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

            // Issue the callback request using HTMX.
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
                    }, 300); // Match CSS animation duration.
                }, delay);
            });
        });

        console.log('hxcmp extension loaded');
    }

    // Wait for DOM to be ready before attaching event listeners.
    // This ensures document.body exists when the script is in <head>.
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        // DOM already loaded (script is deferred or at end of body)
        init();
    }
})();
