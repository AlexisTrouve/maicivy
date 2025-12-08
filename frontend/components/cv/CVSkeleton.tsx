export function CVSkeleton() {
  return (
    <div className="space-y-16 animate-pulse">
      {/* Experiences Skeleton */}
      <section>
        <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-64 mb-8" />
        <div className="space-y-8">
          {[1, 2, 3].map((i) => (
            <div key={i} className="ml-20 bg-gray-200 dark:bg-gray-700 rounded-lg h-48" />
          ))}
        </div>
      </section>

      {/* Skills Skeleton */}
      <section>
        <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-48 mb-8" />
        <div className="flex flex-wrap gap-3 p-8">
          {[1, 2, 3, 4, 5, 6, 7, 8].map((i) => (
            <div
              key={i}
              className="h-10 bg-gray-200 dark:bg-gray-700 rounded-full"
              style={{ width: `${80 + Math.random() * 80}px` }}
            />
          ))}
        </div>
      </section>

      {/* Projects Skeleton */}
      <section>
        <div className="h-8 bg-gray-200 dark:bg-gray-700 rounded w-56 mb-8" />
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <div key={i} className="bg-gray-200 dark:bg-gray-700 rounded-lg h-64" />
          ))}
        </div>
      </section>
    </div>
  );
}
