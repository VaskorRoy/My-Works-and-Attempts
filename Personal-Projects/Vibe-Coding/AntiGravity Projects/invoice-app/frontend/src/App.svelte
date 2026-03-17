<script lang="ts">
  import Sidebar from './lib/components/Sidebar.svelte';
  import Dashboard from './routes/Dashboard.svelte';
  import Invoices from './routes/Invoices.svelte';
  import Settings from './routes/Settings.svelte';
  import { ShieldAlert } from 'lucide-svelte';
  import './app.css';

  let currentRoute = 'dashboard';

  let requiredPin = localStorage.getItem('app_pin') || '';
  let isLocked = !!requiredPin;
  let pinInput = '';
  let pinError = false;

  function unlock() {
    if (pinInput === requiredPin) {
      isLocked = false;
      pinError = false;
    } else {
      pinError = true;
      pinInput = '';
    }
  }
</script>

<main class="flex h-screen w-screen font-sans antialiased overflow-hidden">
  
  <!-- Navigation Sidebar -->
  <Sidebar bind:currentRoute />

  <!-- Main Content Area -->
  <section class="flex-1 flex flex-col relative overflow-y-auto">
    <!-- Topbar Header -->
    <header class="h-16 border-b border-surfaceAcc bg-surface/50 backdrop-blur-md sticky top-0 z-10 flex items-center px-8 shadow-sm">
      <h1 class="text-xl font-bold tracking-tight text-textMain/90 capitalize w-full">{currentRoute}</h1>
    </header>

    <!-- Dynamic View Details -->
    <div class="p-8 w-full max-w-7xl mx-auto flex-1">
      {#if currentRoute === 'dashboard'}
        <Dashboard />
      {:else if currentRoute === 'invoices'}
        <Invoices />
      {:else if currentRoute === 'settings'}
        <Settings />
      {/if}
    </div>
  </section>

</main>

{#if isLocked}
<div class="fixed inset-0 z-[100] bg-background flex items-center justify-center p-4">
  <div class="bg-surface border border-surfaceAcc shadow-2xl rounded-2xl p-8 max-w-sm w-full flex flex-col items-center">
    <div class="w-16 h-16 bg-primary/10 text-primary rounded-full flex items-center justify-center mb-6">
      <ShieldAlert size={32} />
    </div>
    <h1 class="text-xl font-bold mb-2">App Locked</h1>
    <p class="text-sm text-gray-400 text-center mb-6">Please enter your PIN to continue.</p>
    
    <input 
      type="password" 
      bind:value={pinInput} 
      on:keydown={(e) => e.key === 'Enter' && unlock()}
      placeholder="••••"
      class="w-full text-center tracking-[1em] font-mono text-2xl bg-background border {pinError ? 'border-danger focus:border-danger' : 'border-surfaceAcc focus:border-primary'} rounded-xl py-3 outline-none transition-colors mb-4"
    />
    
    {#if pinError}
      <p class="text-xs text-danger mb-4">Incorrect PIN. Try again.</p>
    {/if}

    <button on:click={unlock} class="w-full bg-primary hover:bg-primary/90 text-white font-medium py-3 rounded-xl transition-colors">
      Unlock
    </button>
  </div>
</div>
{/if}
