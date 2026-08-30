import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { toast } from 'vue-sonner';

export const useUiStore = defineStore('ui', () => {
  // State
  const isDarkMode = ref(false);
  const isSidebarOpen = ref(true);
  const isLoading = ref(false);

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
    // Delegate to the global vue-sonner toast API. This works across Vue
    // islands because vue-sonner uses a module-level singleton that renders
    // into the first mounted <Toaster/> it finds in the DOM.
    const opts = { duration } as const;
    switch (type) {
      case 'success':
        toast.success(message, opts);
        break;
      case 'error':
        toast.error(message, opts);
        break;
      case 'warning':
        toast.warning(message, opts);
        break;
      default:
        toast.info(message, opts);
    }
  }

  function removeToast(id: string) {
    if (id) toast.dismiss(id);
  }

  return {
    // State
    isDarkMode,
    isSidebarOpen,
    isLoading,
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
