import textureBand from '@/assets/texture-band.png';
import { Catalog } from '@/components/shop/catalog';
import { shop } from '@/components/shop/config';

// Home is the public shop front: masthead, hero, the catalog grid, and where
// to buy. One page — there is nowhere else to go yet.
export default function HomePage() {
  return (
    <div className="flex min-h-screen flex-col bg-parlor-paper text-parlor-ink">
      <Masthead />

      <main className="grow">
        <Hero />
        <Ornament />
        <Catalog />
        <WhereToBuy />
      </main>

      <Footer />
    </div>
  );
}

// The logo is gold line art on transparency, so it sits straight on the dark
// band with nothing behind it — the band supplies the ground the mark needs.
function Masthead() {
  return (
    <header className="border-b border-parlor-rule bg-parlor-ink">
      <div className="mx-auto flex max-w-6xl flex-col items-center gap-3 px-6 py-8 text-center">
        <a
          href="#top"
          aria-label={shop.name}
          className="transition-opacity hover:opacity-85"
        >
          <img
            src="/logo.png"
            alt={shop.name}
            width={446}
            height={437}
            className="block h-48 w-auto sm:h-56"
          />
        </a>
        <p className="text-[0.62rem] tracking-[0.32em] text-parlor-paper/70 uppercase sm:text-[0.7rem]">
          {shop.tagline}
        </p>
      </div>
    </header>
  );
}

function Hero() {
  return (
    <section
      id="top"
      className="border-b border-parlor-rule bg-[radial-gradient(circle_at_50%_0%,#fdf9f0,transparent_70%)]"
    >
      <div className="mx-auto flex max-w-3xl flex-col items-center gap-6 px-6 py-20 text-center">
        <p className="text-[0.62rem] tracking-[0.32em] text-parlor-brass uppercase">
          One of one, every time
        </p>
        <h1 className="font-display text-4xl leading-[1.15] text-parlor-ink sm:text-5xl">
          Old things, kept well
        </h1>
        <p className="max-w-xl text-[0.95rem] leading-relaxed text-parlor-wood">
          {shop.intro}
        </p>
        <div className="mt-2 flex flex-wrap items-center justify-center gap-3">
          <a
            href="#collection"
            className="border border-parlor-wood bg-parlor-wood px-7 py-3 text-[0.7rem] tracking-[0.18em] text-parlor-paper uppercase transition-colors hover:bg-parlor-ink"
          >
            Browse the collection
          </a>
          <a
            href={shop.whatnot.url}
            target="_blank"
            rel="noreferrer"
            className="border border-parlor-brass px-7 py-3 text-[0.7rem] tracking-[0.18em] text-parlor-ink uppercase transition-colors hover:bg-parlor-brass/15"
          >
            Watch on Whatnot
          </a>
        </div>
      </div>
    </section>
  );
}

// Ornament is the decorative rule between sections — the one piece of
// shop-sign flourish on the page.
function Ornament() {
  return (
    <div
      aria-hidden
      className="mx-auto flex max-w-6xl items-center gap-4 px-6 pt-12 text-parlor-brass"
    >
      <span className="h-px grow bg-parlor-rule" />
      <span className="text-xs tracking-[0.5em]">❖ ❖ ❖</span>
      <span className="h-px grow bg-parlor-rule" />
    </div>
  );
}

function WhereToBuy() {
  return (
    <section
      id="where-to-buy"
      className="border-t border-parlor-rule bg-parlor-paper-dim"
    >
      <div className="mx-auto flex max-w-3xl flex-col items-center gap-5 px-6 py-16 text-center">
        <p className="text-[0.62rem] tracking-[0.32em] text-parlor-muted uppercase">
          Where to buy
        </p>
        <h2 className="font-display text-3xl text-parlor-ink">
          We sell live on Whatnot
        </h2>
        <p className="max-w-xl text-sm leading-relaxed text-parlor-wood">
          {shop.whatnot.blurb}
        </p>
        <a
          href={shop.whatnot.url}
          target="_blank"
          rel="noreferrer"
          className="border border-parlor-wood bg-parlor-wood px-7 py-3 text-[0.7rem] tracking-[0.18em] text-parlor-paper uppercase transition-colors hover:bg-parlor-ink"
        >
          Follow @{shop.whatnot.handle}
        </a>
      </div>
    </section>
  );
}

function Footer() {
  return (
    <footer className="bg-parlor-ink text-parlor-paper">
      <div
        aria-hidden
        className="h-[46px] bg-[length:auto_100%] bg-repeat-x opacity-80"
        style={{ backgroundImage: `url(${textureBand})` }}
      />
      <div className="mx-auto flex max-w-6xl flex-col items-center gap-2 px-6 py-8 text-center">
        <p className="font-display text-lg">{shop.name}</p>
        <p className="text-[0.62rem] tracking-[0.28em] uppercase opacity-70">
          {shop.tagline}
        </p>
      </div>
    </footer>
  );
}
