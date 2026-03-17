<script lang="ts">
  import { LayoutDashboard, FileText, Settings as SettingsIcon, Moon, Sun } from 'lucide-svelte';
  import { onMount } from 'svelte';

  export let currentRoute: string;
  let isDarkMode = true;

  onMount(() => {
    // Check local storage or system pref on load
    const saved = localStorage.getItem('theme');
    if (saved) {
      isDarkMode = saved === 'dark';
    } else {
      isDarkMode = window.matchMedia('(prefers-color-scheme: dark)').matches;
    }
    applyTheme();
  });

  function toggleTheme() {
    isDarkMode = !isDarkMode;
    localStorage.setItem('theme', isDarkMode ? 'dark' : 'light');
    applyTheme();
  }

  function applyTheme() {
    if (isDarkMode) {
      document.documentElement.removeAttribute('data-theme');
    } else {
      document.documentElement.setAttribute('data-theme', 'light');
    }
  }

  const routes = [
    { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { id: 'invoices', label: 'Invoices', icon: FileText },
    { id: 'settings', label: 'Settings', icon: SettingsIcon },
  ];
</script>

<aside class="w-64 bg-surface border-r border-surfaceAcc shadow-lg flex flex-col h-full z-20">
  <div class="h-16 flex items-center px-6 border-b border-surfaceAcc/50 text-xl font-black text-textMain tracking-widest gap-2">
    <div class="w-8 h-8 rounded-lg bg-gradient-to-tr from-primary to-secondary flex items-center justify-center text-sm shadow-md">
      <FileText size={18} class="text-white"/>
    </div>
    Invoice Pro
  </div>

  <nav class="flex-1 px-4 py-6 space-y-2">
    {#each routes as route}
      <button
        class="w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200 text-left cursor-pointer
        {currentRoute === route.id ? 'bg-primary text-white shadow-md shadow-primary/20 scale-[1.02]' : 'text-textMuted hover:bg-surfaceAcc hover:text-textMain'}"
        on:click={() => currentRoute = route.id}
      >
        <svelte:component this={route.icon} size={20} />
        <span class="font-medium text-sm">{route.label}</span>
      </button>
    {/each}
  </nav>

  <div class="p-6 border-t border-surfaceAcc/50 mt-auto space-y-4">
    <button on:click={toggleTheme} class="w-full flex items-center justify-center gap-2 py-2.5 rounded-xl border border-surfaceAcc text-textMuted hover:text-textMain hover:bg-surfaceAcc/50 transition-colors text-sm font-medium">
      {#if isDarkMode}
        <Sun size={16} /> Light Mode
      {:else}
        <Moon size={16} /> Dark Mode
      {/if}
    </button>
    <div class="text-xs text-textMuted text-center font-medium">Invoice MVP v1.1.0</div>
  </div>
</aside>
