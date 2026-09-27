import React, { Suspense } from 'react';
import SearchResults from './SearchResults';

const SearchLoading = () => {
    return <p className="text-center p-12 text-muted-foreground font-medium">Loading search results...</p>;
};

const SearchPage = () => {
  return (
    <Suspense fallback={<SearchLoading />}>
      <SearchResults />
    </Suspense>
  );
};

export default SearchPage;