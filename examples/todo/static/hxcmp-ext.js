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
