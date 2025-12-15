import { Card, CardBody } from './_components/ui/Card';
import { SkeletonLine } from './_components/ui/Skeleton';

export default function Loading() {
  return (
    <main className="container">
      <Card>
        <CardBody>
          <div className="stack">
            <SkeletonLine width="240px" />
            <SkeletonLine width="520px" />
          </div>
        </CardBody>
      </Card>
    </main>
  );
}
