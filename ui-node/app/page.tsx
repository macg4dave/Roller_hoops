import Link from 'next/link';

export default function HomePage() {
  return (
    <main>
      <h1>Roller_hoops</h1>
      <p>Go + Node + Postgres network tracker (UI service).</p>
      <ul>
        <li>
          <Link href="/devices">Devices</Link>
        </li>
      </ul>
    </main>
  );
}
