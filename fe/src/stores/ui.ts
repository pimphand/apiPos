import { defineStore } from 'pinia';
import { ref, computed } from 'vue';

export const useUiStore = defineStore('ui', () => {
  // State
  const isDarkMode = ref(false);
  const isSidebarOpen = ref(true);
  const isLoading = ref(false);
  const toasts = ref<Array<{
    id: string;
    type: 'success' | 'error' | 'warning' | 'info';
    message: string;
    duration?: number;
  }>>([]);

  // Getters
  const theme = computed(() => (isDarkMode.value ? 'dark' : 'light'));

  // Actions
  function toggleDarkMode() {
    isDarkMode.value = !isDarkMode.value;
    document.documentElement.classList.toggle('dark', isDarkMode.value);
    localStorage.setItem('theme', theme.value);
  }

  function setDarkMode(value: boolean) {
    isDarkMode.value = value;
    document.documentElement.classList.toggle('dark', value);
  }

  function loadTheme() {
    const savedTheme = localStorage.getItem('theme');
    if (savedTheme === 'dark') {
      setDarkMode(true);
    } else if (savedTheme === 'light') {
      setDarkMode(false);
    } else {
      // Check system preference
      const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      setDarkMode(prefersDark);
    }
  }

  function toggleSidebar() {
    isSidebarOpen.value = !isSidebarOpen.value;
  }

  function setLoading(value: boolean) {
    isLoading.value = value;
  }

  function addToast(
    type: 'success' | 'error' | 'warning' | 'info',
    message: string,
    duration = 5000
  ) {
    const id = Date.now().toString();
    toasts.value.push({ id, type, message, duration });

    if (duration > 0) {
      setTimeout(() => {
        removeToast(id);
      }, duration);
    }
  }

  function removeToast(id: string) {
    const index = toasts.value.findIndex((t) => t.id === id);
    if (index > -1) {
      toasts.value.splice(index, 1);
    }
  }

  return {
    // State
    isDarkMode,
    isSidebarOpen,
    isLoading,
    toasts,
    // Getters
    theme,
    // Actions
    toggleDarkMode,
    setDarkMode,
    loadTheme,
    toggleSidebar,
    setLoading,
    addToast,
    removeToast,
  };
});
