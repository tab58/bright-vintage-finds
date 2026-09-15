// Shop-front copy. Hardcoded on purpose: none of it lives in the database yet.
// TODO(owner): replace the placeholders below with the real details.
export const shop = {
  name: 'Bright Vintage Finds',
  tagline: 'Curated antiques & vintage goods',
  // The hero blurb, two or three sentences at most.
  intro:
    'A small, always-changing collection of antique and vintage pieces — glassware, furniture, lighting and odd treasures — hunted down one estate sale at a time and photographed as they are.',
  // Whatnot is the only place to buy right now.
  whatnot: {
    handle: 'brightvintagefinds',
    get url() {
      return `https://www.whatnot.com/user/${this.handle}`;
    },
    blurb:
      'Everything here sells live on Whatnot. Follow the shop to get a ping when the next show starts, and bid on the pieces you have been watching.',
  },
} as const;
