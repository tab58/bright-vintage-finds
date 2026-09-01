import * as stylex from '@stylexjs/stylex'
import textureBand from './assets/texture-band.png'
import poster from './assets/card-bright.png'

export default function App() {
  return (
    <div {...stylex.props(styles.page)}>
      <div
        {...stylex.props(styles.band)}
        style={{ backgroundImage: `url(${textureBand})` }}
      />
      <main {...stylex.props(styles.center)}>
        <img
          src={poster}
          alt="Up 'n' Bright vintage poster art"
          {...stylex.props(styles.poster)}
        />
        <h1 {...stylex.props(styles.title)}>Bright Vintage Finds</h1>
        <p {...stylex.props(styles.comingSoon)}>Full site coming soon</p>
      </main>
      <div
        {...stylex.props(styles.band, styles.bandBottom)}
        style={{ backgroundImage: `url(${textureBand})` }}
      />
    </div>
  )
}

const styles = stylex.create({
  page: {
    minHeight: '100vh',
    backgroundColor: '#2e2418',
    display: 'flex',
    flexDirection: 'column',
    boxSizing: 'border-box',
  },
  band: {
    height: '70px',
    backgroundRepeat: 'repeat-x',
    backgroundSize: 'auto 100%',
  },
  bandBottom: {
    transform: 'scaleY(-1)',
  },
  center: {
    flexGrow: 1,
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    textAlign: 'center',
    paddingBlock: '48px',
    paddingInline: '24px',
  },
  poster: {
    width: 'min(300px, 80vw)',
    height: 'auto',
    borderRadius: '10px',
    transform: 'rotate(-2deg)',
    boxShadow: '0 12px 40px rgba(0, 0, 0, 0.6)',
  },
  title: {
    fontFamily: "Georgia, 'Times New Roman', serif",
    fontSize: '2.2rem',
    fontWeight: 700,
    color: '#e8d9b8',
    marginTop: '36px',
    marginBottom: '0',
  },
  comingSoon: {
    fontFamily:
      "-apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif",
    fontSize: '0.85rem',
    letterSpacing: '0.14em',
    textTransform: 'uppercase',
    color: '#a3927c',
    marginTop: '12px',
    marginBottom: '0',
  },
})
