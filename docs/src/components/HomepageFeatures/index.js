import clsx from "clsx";
import Heading from "@theme/Heading";
import styles from "./styles.module.css";

const FeatureList = [
  {
    title:
      "Physically backup your PostgreSQL Clusters deployed with CloudNativePG.",
    Svg: require("@site/static/img/backup.svg").default,
    description: <> </>,
  },
  {
    title:
      "Powered by the most reliable PostgreSQL Backup & Restore tool, pgBackRest !",
    Svg: require("@site/static/img/tool.svg").default,
    description: <></>,
  },
  {
    title: "Point In Time Recovery is supported in recovery operations.",
    Svg: require("@site/static/img/pitr.svg").default,
    description: <></>,
  },
];

function Feature({ Svg, title, description }) {
  return (
    <div className={clsx("col col--4")}>
      <div className="text--center">
        <Svg className={styles.featureSvg} role="img" />
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures() {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
