import { Card, CardBody } from '../_components/ui/Card';
import { SkeletonLine } from '../_components/ui/Skeleton';

export default function Loading() {
  return (
    <div className="stack">
      <Card>
        <CardBody>
          <div className="stack">
            <SkeletonLine width="240px" />
            <SkeletonLine width="520px" />
          </div>
        </CardBody>
      </Card>

      <div className="devicesDashboardTop">
        <Card>
          <CardBody>
            <div className="stack">
              <SkeletonLine width="180px" />
              <SkeletonLine width="260px" />
              <SkeletonLine width="220px" />
            </div>
          </CardBody>
        </Card>

        <Card>
          <CardBody>
            <div className="stack">
              <SkeletonLine width="180px" />
              <SkeletonLine width="260px" />
              <SkeletonLine width="220px" />
            </div>
          </CardBody>
        </Card>
      </div>
    </div>
  );
}
