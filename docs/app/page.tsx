import Link from 'next/link';

const features = [
  {
    title: 'Network Management',
    description: 'Start, stop, and manage local and remote Lux networks',
    href: '/docs/network',
    icon: '🌐',
  },
  {
    title: 'Blockchain Creation',
    description: 'Create and deploy custom blockchains with any VM',
    href: '/docs/blockchain',
    icon: '⛓️',
  },
  {
    title: 'Validator Operations',
    description: 'Add, remove, and manage validator nodes',
    href: '/docs/validators',
    icon: '🔐',
  },
  {
    title: 'Wallet & Keys',
    description: 'Manage wallets, keys, and transactions',
    href: '/docs/wallet',
    icon: '👛',
  },
  {
    title: 'Sovereign L1s',
    description: 'Create quantum-safe sovereign L1 chains',
    href: '/docs/subnet',
    icon: '🛡️',
  },
  {
    title: 'Testing Tools',
    description: 'Local simulation and development tools',
    href: '/docs/testing',
    icon: '🧪',
  },
];

const commands = [
  { cmd: 'lux network start', desc: 'Start local 5-node network' },
  { cmd: 'lux blockchain create mychain', desc: 'Create new blockchain' },
  { cmd: 'lux blockchain deploy mychain', desc: 'Deploy to network' },
  { cmd: 'lux validator add', desc: 'Add validator node' },
];

export default function Home() {
  return (
    <main className="min-h-screen">
      {/* Hero Section */}
      <section className="relative overflow-hidden bg-gradient-to-b from-fd-background to-fd-muted/30">
        <div className="absolute inset-0 bg-grid-white/[0.02] bg-[size:60px_60px]" />
        <div className="relative mx-auto max-w-7xl px-6 py-24 sm:py-32 lg:px-8">
          <div className="mx-auto max-w-2xl text-center">
            <div className="mb-8 flex justify-center">
              <div className="rounded-full bg-fd-primary/10 px-4 py-1.5 text-sm font-medium text-fd-primary ring-1 ring-inset ring-fd-primary/20">
                v1.0.0 — Production Ready
              </div>
            </div>
            <h1 className="text-4xl font-bold tracking-tight sm:text-6xl bg-gradient-to-r from-fd-foreground to-fd-foreground/70 bg-clip-text text-transparent">
              Lux CLI
            </h1>
            <p className="mt-6 text-lg leading-8 text-fd-muted-foreground">
              The official command-line interface for the Lux Network.
              Manage networks, validators, blockchains, and wallets.
            </p>
            <div className="mt-10 flex items-center justify-center gap-x-4">
              <Link
                href="/docs"
                className="rounded-lg bg-fd-primary px-5 py-2.5 text-sm font-semibold text-fd-primary-foreground shadow-sm hover:bg-fd-primary/90 transition-colors"
              >
                Get Started
              </Link>
              <Link
                href="https://github.com/luxfi/cli"
                className="rounded-lg px-5 py-2.5 text-sm font-semibold text-fd-foreground ring-1 ring-fd-border hover:bg-fd-muted transition-colors"
              >
                GitHub →
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Quick Install */}
      <section className="border-y border-fd-border bg-fd-muted/30">
        <div className="mx-auto max-w-7xl px-6 py-12 lg:px-8">
          <div className="mx-auto max-w-2xl">
            <h2 className="text-xl font-bold tracking-tight mb-4">Quick Install</h2>
            <div className="rounded-xl border border-fd-border bg-fd-card overflow-hidden">
              <div className="border-b border-fd-border px-4 py-2 bg-fd-muted/50">
                <span className="text-sm text-fd-muted-foreground">Terminal</span>
              </div>
              <pre className="p-4 text-sm overflow-x-auto">
                <code className="text-fd-foreground">{`# Install via script
curl -sSfL https://raw.githubusercontent.com/luxfi/cli/main/scripts/install.sh | sh

# Add to PATH
export PATH=$PATH:~/.lux/bin

# Verify installation
lux --version`}</code>
              </pre>
            </div>
          </div>
        </div>
      </section>

      {/* Features Grid */}
      <section className="mx-auto max-w-7xl px-6 py-24 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <h2 className="text-3xl font-bold tracking-tight sm:text-4xl">
            Everything you need
          </h2>
          <p className="mt-4 text-lg text-fd-muted-foreground">
            Complete tooling for Lux blockchain development
          </p>
        </div>
        <div className="mx-auto mt-16 grid max-w-5xl grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((feature) => (
            <Link
              key={feature.title}
              href={feature.href}
              className="group relative rounded-xl border border-fd-border bg-fd-card p-6 hover:border-fd-primary/50 hover:bg-fd-muted/50 transition-all"
            >
              <div className="text-3xl mb-4">{feature.icon}</div>
              <h3 className="text-lg font-semibold text-fd-foreground group-hover:text-fd-primary transition-colors">
                {feature.title}
              </h3>
              <p className="mt-2 text-sm text-fd-muted-foreground">
                {feature.description}
              </p>
            </Link>
          ))}
        </div>
      </section>

      {/* Common Commands */}
      <section className="border-t border-fd-border bg-fd-muted/30">
        <div className="mx-auto max-w-7xl px-6 py-24 lg:px-8">
          <div className="mx-auto max-w-2xl">
            <h2 className="text-2xl font-bold tracking-tight mb-8">Common Commands</h2>
            <div className="space-y-4">
              {commands.map((item) => (
                <div key={item.cmd} className="rounded-lg border border-fd-border bg-fd-card p-4">
                  <code className="text-sm font-mono text-fd-primary">{item.cmd}</code>
                  <p className="mt-1 text-sm text-fd-muted-foreground">{item.desc}</p>
                </div>
              ))}
            </div>
            <div className="mt-8 flex gap-4">
              <Link
                href="/docs/commands"
                className="text-sm font-medium text-fd-primary hover:text-fd-primary/80"
              >
                View all commands →
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t border-fd-border">
        <div className="mx-auto max-w-7xl px-6 py-12 lg:px-8">
          <div className="flex flex-col items-center justify-between gap-4 sm:flex-row">
            <p className="text-sm text-fd-muted-foreground">
              © 2025 Lux Partners. MIT License.
            </p>
            <div className="flex gap-6">
              <Link href="https://github.com/luxfi/cli" className="text-sm text-fd-muted-foreground hover:text-fd-foreground">
                GitHub
              </Link>
              <Link href="https://discord.gg/lux" className="text-sm text-fd-muted-foreground hover:text-fd-foreground">
                Discord
              </Link>
              <Link href="https://lux.network" className="text-sm text-fd-muted-foreground hover:text-fd-foreground">
                Lux Network
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </main>
  );
}
