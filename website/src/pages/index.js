import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Heading from '@theme/Heading';
import styles from './index.module.css';

const modes = [
  {
    title: 'User Guide',
    description:
      "For applications built against the SDK: installation, authentication, the fluent request builder pattern, and error handling.",
    to: '/docs/',
    cta: 'Read the User Guide',
  },
  {
    title: 'Contributor Guide',
    description:
      'For working on the SDK itself: repo architecture, regenerating the client from the Omada API description, the BDD suite, and CI/release.',
    to: '/contributing/',
    cta: 'Read the Contributor Guide',
  },
];

function ModeCard({title, description, to, cta}) {
  return (
    <div className={styles.modeCard}>
      <Heading as="h2">{title}</Heading>
      <p>{description}</p>
      <Link className="button button--primary button--lg" to={to}>
        {cta}
      </Link>
    </div>
  );
}

export default function Home() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout title={siteConfig.title} description={siteConfig.tagline}>
      <header className={styles.heroBanner}>
        <div className="container">
          <Heading as="h1" className="hero__title">
            {siteConfig.title}
          </Heading>
          <p className="hero__subtitle">{siteConfig.tagline}</p>
        </div>
      </header>
      <main className="container">
        <div className={styles.modeGrid}>
          {modes.map((mode) => (
            <ModeCard key={mode.to} {...mode} />
          ))}
        </div>
      </main>
    </Layout>
  );
}
