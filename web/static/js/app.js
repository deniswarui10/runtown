// EventHub JavaScript Application

// Initialize Alpine.js components
document.addEventListener('alpine:init', () => {
    // Global Alpine data
    Alpine.data('dropdown', () => ({
        open: false,
        toggle() {
            this.open = !this.open;
        },
        close() {
            this.open = false;
        }
    }));

    Alpine.data('modal', () => ({
        open: false,
        show() {
            this.open = true;
            document.body.style.overflow = 'hidden';
        },
        hide() {
            this.open = false;
            document.body.style.overflow = 'auto';
        }
    }));

    Alpine.data('toast', () => ({
        visible: false,
        message: '',
        type: 'info',
        show(message, type = 'info') {
            this.message = message;
            this.type = type;
            this.visible = true;
            setTimeout(() => {
                this.hide();
            }, 5000);
        },
        hide() {
            this.visible = false;
        }
    }));
});

// HTMX Configuration
document.addEventListener('DOMContentLoaded', function() {
    // Configure HTMX
    htmx.config.globalViewTransitions = true;
    htmx.config.scrollBehavior = 'smooth';
    
    // Add loading indicators
    document.body.addEventListener('htmx:beforeRequest', function(evt) {
        const target = evt.target;
        if (target.classList.contains('btn')) {
            const originalText = target.textContent;
            target.setAttribute('data-original-text', originalText);
            target.innerHTML = '<span class="htmx-indicator">Loading...</span>';
            target.disabled = true;
        }
    });

    document.body.addEventListener('htmx:afterRequest', function(evt) {
        const target = evt.target;
        if (target.classList.contains('btn') && target.hasAttribute('data-original-text')) {
            target.textContent = target.getAttribute('data-original-text');
            target.removeAttribute('data-original-text');
            target.disabled = false;
        }
    });

    // Handle form validation errors
    document.body.addEventListener('htmx:responseError', function(evt) {
        if (evt.detail.xhr.status === 422) {
            // For 422 errors, HTMX will automatically swap the content
            // The server returns the form with validation errors already rendered
            // No need to parse JSON - the HTML response contains the errors
            console.log('Form validation failed - errors are displayed in the form');
        }
    });

    // Auto-hide alerts after 5 seconds
    const alerts = document.querySelectorAll('[data-auto-hide]');
    alerts.forEach(alert => {
        setTimeout(() => {
            alert.style.opacity = '0';
            setTimeout(() => {
                alert.remove();
            }, 300);
        }, 5000);
    });
});

// Utility Functions
function showValidationErrors(errors) {
    // Clear existing errors
    document.querySelectorAll('.error-message').forEach(el => el.remove());
    
    // Show new errors
    Object.keys(errors).forEach(field => {
        const input = document.querySelector(`[name="${field}"]`);
        if (input) {
            input.classList.add('border-red-300', 'focus:ring-red-500');
            
            const errorDiv = document.createElement('div');
            errorDiv.className = 'error-message text-sm text-red-600 mt-1';
            errorDiv.textContent = errors[field][0];
            
            input.parentNode.appendChild(errorDiv);
        }
    });
}

function clearValidationErrors() {
    document.querySelectorAll('.error-message').forEach(el => el.remove());
    document.querySelectorAll('input, textarea, select').forEach(el => {
        el.classList.remove('border-red-300', 'focus:ring-red-500');
    });
}

function showToast(message, type = 'info') {
    const toast = document.createElement('div');
    toast.className = `fixed top-4 right-4 z-50 p-4 rounded-lg shadow-lg max-w-sm fade-in ${getToastClasses(type)}`;
    toast.innerHTML = `
        <div class="flex items-center">
            <div class="flex-shrink-0">
                ${getToastIcon(type)}
            </div>
            <div class="ml-3">
                <p class="text-sm font-medium">${message}</p>
            </div>
            <div class="ml-4 flex-shrink-0">
                <button onclick="this.parentElement.parentElement.parentElement.remove()" class="text-gray-400 hover:text-gray-600">
                    <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                    </svg>
                </button>
            </div>
        </div>
    `;
    
    document.body.appendChild(toast);
    
    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => {
            toast.remove();
        }, 300);
    }, 5000);
}

function getToastClasses(type) {
    switch (type) {
        case 'success':
            return 'bg-green-50 border border-green-200 text-green-800';
        case 'error':
            return 'bg-red-50 border border-red-200 text-red-800';
        case 'warning':
            return 'bg-yellow-50 border border-yellow-200 text-yellow-800';
        default:
            return 'bg-blue-50 border border-blue-200 text-blue-800';
    }
}

function getToastIcon(type) {
    switch (type) {
        case 'success':
            return '<svg class="h-5 w-5 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>';
        case 'error':
            return '<svg class="h-5 w-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>';
        case 'warning':
            return '<svg class="h-5 w-5 text-yellow-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 16c-.77.833.192 2.5 1.732 2.5z"></path></svg>';
        default:
            return '<svg class="h-5 w-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>';
    }
}

// Ticket selection functions
function updateQuantity(button, change) {
    const input = button.parentElement.querySelector('input[type="number"]');
    const currentValue = parseInt(input.value) || 0;
    const newValue = Math.max(0, Math.min(currentValue + change, parseInt(input.max)));
    input.value = newValue;
    calculateTotal();
}

function calculateTotal() {
    let total = 0;
    const ticketInputs = document.querySelectorAll('input[name^="tickets["]');
    
    ticketInputs.forEach(input => {
        const quantity = parseInt(input.value) || 0;
        const price = parseInt(input.dataset.price) || 0;
        total += quantity * price;
    });
    
    const totalElement = document.getElementById('total-price');
    if (totalElement) {
        totalElement.textContent = 'KSh ' + (total / 100).toFixed(2);
    }
}

// Form Enhancement
function enhanceForm(form) {
    const inputs = form.querySelectorAll('input, textarea, select');
    
    inputs.forEach(input => {
        // Clear errors on input
        input.addEventListener('input', clearValidationErrors);
        
        // Add floating label effect
        if (input.type !== 'checkbox' && input.type !== 'radio') {
            input.addEventListener('focus', function() {
                this.parentNode.classList.add('focused');
            });
            
            input.addEventListener('blur', function() {
                if (!this.value) {
                    this.parentNode.classList.remove('focused');
                }
            });
        }
    });
}

// Initialize forms on page load
document.addEventListener('DOMContentLoaded', function() {
    document.querySelectorAll('form').forEach(enhanceForm);
});

// Handle dynamic content
document.body.addEventListener('htmx:afterSwap', function(evt) {
    // Re-initialize forms in new content
    evt.detail.target.querySelectorAll('form').forEach(enhanceForm);
    
    // Re-initialize Alpine components
    if (window.Alpine) {
        Alpine.initTree(evt.detail.target);
    }
});

// Keyboard shortcuts
document.addEventListener('keydown', function(evt) {
    // Escape key closes modals and dropdowns
    if (evt.key === 'Escape') {
        document.querySelectorAll('[x-data]').forEach(el => {
            if (el._x_dataStack && el._x_dataStack[0].open) {
                el._x_dataStack[0].open = false;
            }
        });
    }
    
    // Ctrl/Cmd + K opens search
    if ((evt.ctrlKey || evt.metaKey) && evt.key === 'k') {
        evt.preventDefault();
        const searchInput = document.querySelector('input[name="q"]');
        if (searchInput) {
            searchInput.focus();
        }
    }
});

// Smooth scrolling for anchor links
document.addEventListener('click', function(evt) {
    if (evt.target.matches('a[href^="#"]')) {
        evt.preventDefault();
        const target = document.querySelector(evt.target.getAttribute('href'));
        if (target) {
            target.scrollIntoView({
                behavior: 'smooth',
                block: 'start'
            });
        }
    }
});

// Image lazy loading fallback
document.addEventListener('DOMContentLoaded', function() {
    const images = document.querySelectorAll('img[data-src]');
    
    if ('IntersectionObserver' in window) {
        const imageObserver = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    img.src = img.dataset.src;
                    img.classList.remove('lazy');
                    imageObserver.unobserve(img);
                }
            });
        });
        
        images.forEach(img => imageObserver.observe(img));
    } else {
        // Fallback for older browsers
        images.forEach(img => {
            img.src = img.dataset.src;
        });
    }
});

// Export functions for global use
window.EventHub = {
    showToast,
    showValidationErrors,
    clearValidationErrors,
    enhanceForm,
    updateQuantity,
    calculateTotal
};